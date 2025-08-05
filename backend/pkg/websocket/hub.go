// backend/pkg/websocket/hub.go
package websocket

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ripple/pkg/auth"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Maps user IDs to clients
	userClients map[int]*Client

	// Maps group IDs to group members' clients
	groupClients map[int]map[*Client]bool

	// Inbound messages for all clients
	broadcast chan []byte

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Stop signal
	stop chan struct{}

	// Database connection
	db *sql.DB

	// Mutex for concurrent access
	mu sync.RWMutex
}

// Client represents a WebSocket client
type Client struct {
	// The WebSocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// User ID of the client
	userID int

	// Hub reference
	hub *Hub

	// Last seen timestamp
	lastSeen time.Time

	// User groups (cached for quick access)
	userGroups map[int]bool
}

// Message types for WebSocket communication
type MessageType string

const (
	MessageTypePrivate          MessageType = "private_message"
	MessageTypeGroup            MessageType = "group_message"
	MessageTypeNotification     MessageType = "notification"
	MessageTypeNotificationAck  MessageType = "notification_ack"
	MessageTypeUserOnline       MessageType = "user_online"
	MessageTypeUserOffline      MessageType = "user_offline"
	MessageTypeTyping           MessageType = "typing"
	MessageTypeReadStatus       MessageType = "read_status"
	MessageTypeDelivered        MessageType = "delivered"
	MessageTypeError            MessageType = "error"
	MessageTypeHeartbeat        MessageType = "heartbeat"
	MessageTypeUserList         MessageType = "user_list"
	MessageTypePresence         MessageType = "presence"
	MessageTypePing             MessageType = "ping"
	MessageTypePong             MessageType = "pong"
	MessageTypeConnectionStatus MessageType = "connection_status"
)

// WebSocket message structure
type WSMessage struct {
	Type      MessageType `json:"type"`
	Content   string      `json:"content,omitempty"`
	From      int         `json:"from,omitempty"`
	To        int         `json:"to,omitempty"`
	GroupID   int         `json:"group_id,omitempty"`
	MessageID int         `json:"message_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Data      any         `json:"data,omitempty"`
}

// NewHub creates a new WebSocket hub
func NewHub(db *sql.DB) *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		userClients:  make(map[int]*Client),
		groupClients: make(map[int]map[*Client]bool),
		broadcast:    make(chan []byte),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		stop:         make(chan struct{}),
		db:           db,
	}
}

// HandleWebSocket upgrades HTTP connection to WebSocket and handles the connection
func HandleWebSocket(hub *Hub, sm *auth.SessionManager, w http.ResponseWriter, r *http.Request) {
	// Use the existing WebSocket authentication function
	userID, err := AuthenticateWebSocket(r, sm)
	if err != nil {
		log.Printf("WebSocket: Authentication failed: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// In production, you should check the origin against your allowed domains
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket: Error upgrading to WebSocket: %v", err)
		return
	}

	client := &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, 256),
		userID:     userID,
		lastSeen:   time.Now(),
		userGroups: make(map[int]bool),
	}

	// Register the client
	client.hub.register <- client

	// Send connection status to client
	go func() {
		time.Sleep(100 * time.Millisecond) // Small delay to ensure registration is complete
		statusMessage := WSMessage{
			Type:      MessageTypeConnectionStatus,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"status":  "connected",
				"user_id": userID,
			},
		}
		client.hub.sendToClient(client, statusMessage)
	}()

	// Start the read and write pumps in separate goroutines
	go client.writePump()
	go client.readPump()
}

// Stop gracefully shuts down the hub
func (h *Hub) Stop() {
	close(h.stop)
}

// Run starts the hub and handles client connections
func (h *Hub) Run() {
	// Start cleanup routine
	go h.cleanupRoutine()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			if client.userID > 0 {
				// If user already has a connection, close the old one
				if oldClient, exists := h.userClients[client.userID]; exists {
					close(oldClient.send)
					delete(h.clients, oldClient)
				}
				h.userClients[client.userID] = client
			}

			// Load user's groups and subscribe to group channels
			h.loadUserGroups(client)

			h.mu.Unlock()

			log.Printf("WebSocket: User %d connected", client.userID)

			// Send initial presence data
			h.sendInitialPresenceData(client)

			// Notify contacts that user is online
			h.notifyUserOnline(client.userID)

			// Send queued messages if any
			go h.sendQueuedMessages(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.userClients, client.userID)

				// Remove from group channels
				for groupID := range client.userGroups {
					if clients, exists := h.groupClients[groupID]; exists {
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.groupClients, groupID)
						}
					}
				}

				close(client.send)
			}
			h.mu.Unlock()

			log.Printf("WebSocket: User %d disconnected", client.userID)

			// Update user presence
			h.updateUserPresence(client.userID, false)

			// Notify contacts that user is offline
			h.notifyUserOffline(client.userID)

		case message := <-h.broadcast:
			// Handle broadcast messages (notifications, etc.)
			var wsMsg WSMessage
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				log.Printf("WebSocket: Error unmarshaling broadcast message: %v", err)
				continue
			}

			h.handleBroadcastMessage(&wsMsg)

		case <-h.stop:
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
		}
	}
}

// sendToClient sends a message to a specific client
func (h *Hub) sendToClient(client *Client, message WSMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("WebSocket: Error marshaling message: %v", err)
		return
	}

	select {
	case client.send <- messageBytes:
		if message.Type != MessageTypePong {
			log.Printf("WebSocket: Message sent to user %d", client.userID)
		}
	default:
		// Client's send channel is full, close the connection
		h.unregisterClient(client)
	}
}

// unregisterClient safely unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	select {
	case h.unregister <- client:
	default:
		// Channel is full, force close
		h.mu.Lock()
		if _, ok := h.clients[client]; ok {
			delete(h.clients, client)
			delete(h.userClients, client.userID)
			close(client.send)
		}
		h.mu.Unlock()
	}
}

// loadUserGroups loads and caches user's group memberships
func (h *Hub) loadUserGroups(client *Client) {
	client.userGroups = make(map[int]bool)

	query := `
		SELECT group_id FROM group_members 
		WHERE user_id = ? AND status = 'accepted'
	`

	rows, err := h.db.Query(query, client.userID)
	if err != nil {
		log.Printf("WebSocket: Error loading user groups: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var groupID int
		if err := rows.Scan(&groupID); err != nil {
			log.Printf("WebSocket: Error scanning group ID: %v", err)
			continue
		}

		client.userGroups[groupID] = true

		// Add client to group channel
		if h.groupClients[groupID] == nil {
			h.groupClients[groupID] = make(map[*Client]bool)
		}
		h.groupClients[groupID][client] = true
	}

	log.Printf("WebSocket: Loaded %d groups for user %d", len(client.userGroups), client.userID)
}

// sendInitialPresenceData sends initial online user list and presence data
func (h *Hub) sendInitialPresenceData(client *Client) {
	// Get user's contacts
	contacts := h.getUserContacts(client.userID)

	// Filter contacts to only online ones
	var onlineContacts []int
	h.mu.RLock()
	for _, contactID := range contacts {
		if _, isOnline := h.userClients[contactID]; isOnline {
			onlineContacts = append(onlineContacts, contactID)
		}
	}
	h.mu.RUnlock()

	// Send initial user list
	userListMsg := WSMessage{
		Type:      MessageTypeUserList,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"online_users": onlineContacts,
		},
	}
	h.sendToClient(client, userListMsg)
}

// updateUserPresence updates user's presence in the database
func (h *Hub) updateUserPresence(userID int, isOnline bool) {
	query := `
		INSERT OR REPLACE INTO user_presence (user_id, last_seen, is_online)
		VALUES (?, ?, ?)
	`

	_, err := h.db.Exec(query, userID, time.Now(), isOnline)
	if err != nil {
		log.Printf("WebSocket: Error updating user presence: %v", err)
	}
}

// notifyUserOnline notifies contacts that a user came online
func (h *Hub) notifyUserOnline(userID int) {
	h.updateUserPresence(userID, true)

	contacts := h.getUserContacts(userID)

	message := WSMessage{
		Type:      MessageTypeUserOnline,
		From:      userID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"user_id": userID,
		},
	}

	messageBytes, _ := json.Marshal(message)

	h.mu.RLock()
	for _, contactID := range contacts {
		if client, exists := h.userClients[contactID]; exists {
			select {
			case client.send <- messageBytes:
			default:
				// Skip if send channel is full
			}
		}
	}
	h.mu.RUnlock()
}

// notifyUserOffline notifies contacts that a user went offline
func (h *Hub) notifyUserOffline(userID int) {
	contacts := h.getUserContacts(userID)

	message := WSMessage{
		Type:      MessageTypeUserOffline,
		From:      userID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"user_id": userID,
		},
	}

	messageBytes, _ := json.Marshal(message)

	h.mu.RLock()
	for _, contactID := range contacts {
		if client, exists := h.userClients[contactID]; exists {
			select {
			case client.send <- messageBytes:
			default:
				// Skip if send channel is full
			}
		}
	}
	h.mu.RUnlock()
}

// getUserContacts gets user's contacts (followers and following)
func (h *Hub) getUserContacts(userID int) []int {
	var contacts []int

	query := `
		SELECT DISTINCT 
			CASE 
				WHEN follower_id = ? THEN following_id 
				ELSE follower_id 
			END as contact_id
		FROM follows 
		WHERE (follower_id = ? OR following_id = ?) AND status = 'accepted'
	`

	rows, err := h.db.Query(query, userID, userID, userID)
	if err != nil {
		log.Printf("WebSocket: Error getting user contacts: %v", err)
		return contacts
	}
	defer rows.Close()

	for rows.Next() {
		var contactID int
		if err := rows.Scan(&contactID); err != nil {
			continue
		}
		contacts = append(contacts, contactID)
	}

	return contacts
}

// handleBroadcastMessage handles broadcast messages
func (h *Hub) handleBroadcastMessage(message *WSMessage) {
	switch message.Type {
	case MessageTypeNotification:
		// Handle notification broadcasts
		if message.To > 0 {
			h.SendNotification(message.To, message.Data)
		}
	}
}

// sendQueuedMessages sends any queued offline messages
func (h *Hub) sendQueuedMessages(client *Client) {
	// Get unread messages for the user
	query := `
		SELECT id, sender_id, content, created_at 
		FROM messages 
		WHERE receiver_id = ? AND read_at IS NULL 
		ORDER BY created_at DESC 
		LIMIT 50
	`

	rows, err := h.db.Query(query, client.userID)
	if err != nil {
		log.Printf("WebSocket: Error getting queued messages: %v", err)
		return
	}
	defer rows.Close()

	var queuedCount int
	for rows.Next() {
		var messageID, senderID int
		var content string
		var createdAt time.Time

		if err := rows.Scan(&messageID, &senderID, &content, &createdAt); err != nil {
			continue
		}

		message := WSMessage{
			Type:      MessageTypePrivate,
			Content:   content,
			From:      senderID,
			To:        client.userID,
			MessageID: messageID,
			Timestamp: createdAt,
		}

		h.sendToClient(client, message)
		queuedCount++
	}

	if queuedCount > 0 {
		log.Printf("WebSocket: Sent %d queued messages to user %d", queuedCount, client.userID)
	}
}

// // SendGroupMessage sends a message to all members of a group
// func (h *Hub) SendGroupMessage(groupID int, senderID int, content string, messageID int) {
// 	message := WSMessage{
// 		Type:      MessageTypeGroup,
// 		Content:   content,
// 		From:      senderID,
// 		GroupID:   groupID,
// 		MessageID: messageID,
// 		Timestamp: time.Now(),
// 	}

// 	h.mu.RLock()
// 	if clients, exists := h.groupClients[groupID]; exists {
// 		for client := range clients {
// 			if client.userID != senderID {
// 				h.sendToClient(client, message)
// 			}
// 		}
// 	}
// 	h.mu.RUnlock()
// }

// // SendPrivateMessage sends a message to a specific user
// func (h *Hub) SendPrivateMessage(senderID int, recipientID int, content string, messageID int) {
// 	message := WSMessage{
// 		Type:      MessageTypePrivate,
// 		Content:   content,
// 		From:      senderID,
// 		To:        recipientID,
// 		MessageID: messageID,
// 		Timestamp: time.Now(),
// 	}

// 	h.mu.RLock()
// 	if client, exists := h.userClients[recipientID]; exists {
// 		h.sendToClient(client, message)

// 		// Send delivery confirmation back to sender
// 		deliveredMsg := WSMessage{
// 			Type:      MessageTypeDelivered,
// 			From:      recipientID,
// 			To:        senderID,
// 			MessageID: messageID,
// 			Timestamp: time.Now(),
// 		}

// 		if senderClient, senderExists := h.userClients[senderID]; senderExists {
// 			h.sendToClient(senderClient, deliveredMsg)
// 		}
// 	}
// 	h.mu.RUnlock()
// }

// IsUserOnline checks if a user is currently online
func (h *Hub) IsUserOnline(userID int) bool {
	h.mu.RLock()
	_, exists := h.userClients[userID]
	h.mu.RUnlock()
	return exists
}

// GetOnlineUsers returns a list of all online user IDs
func (h *Hub) GetOnlineUsers() []int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	onlineUsers := make([]int, 0, len(h.userClients))
	for userID := range h.userClients {
		onlineUsers = append(onlineUsers, userID)
	}
	return onlineUsers
}

// GetOnlineGroupMembers returns online members of a specific group
func (h *Hub) GetOnlineGroupMembers(groupID int) []int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var members []int
	if clients, exists := h.groupClients[groupID]; exists {
		for client := range clients {
			members = append(members, client.userID)
		}
	}
	return members
}

// BroadcastTypingIndicator sends typing status to relevant clients
func (h *Hub) BroadcastTypingIndicator(userID int, chatType string, targetID int, isTyping bool) {
	msg := WSMessage{
		Type:      MessageTypeTyping,
		From:      userID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"user_id":   userID,
			"chat_type": chatType,
			"target_id": targetID,
			"is_typing": isTyping,
		},
	}

	if chatType == "private" {
		msg.To = targetID
	} else if chatType == "group" {
		msg.GroupID = targetID
	}

	messageBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: Error marshaling typing indicator: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if chatType == "private" && targetID > 0 {
		// For private chat, send only to the target user
		if client, exists := h.userClients[targetID]; exists {
			select {
			case client.send <- messageBytes:
			default:
				// Skip if send channel is full
			}
		}
	} else if chatType == "group" && targetID > 0 {
		// For group chat, send to all group members except sender
		if clients, exists := h.groupClients[targetID]; exists {
			for client := range clients {
				if client.userID != userID {
					select {
					case client.send <- messageBytes:
					default:
						// Skip if send channel is full
					}
				}
			}
		}
	}
}

// SendNotification sends a notification to a specific user with retry logic
func (h *Hub) SendNotification(userID int, data interface{}) {
	message := WSMessage{
		Type:      MessageTypeNotification,
		To:        userID,
		Data:      data,
		Timestamp: time.Now(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("WebSocket: Error marshaling notification: %v", err)
		return
	}

	h.mu.RLock()
	if client, exists := h.userClients[userID]; exists {
		select {
		case client.send <- messageBytes:
			log.Printf("WebSocket: Notification sent to user %d", userID)
		default:
			// Channel is full, try to clean up and retry once
			log.Printf("WebSocket: Send channel full for user %d, attempting cleanup", userID)
			go h.handleFullChannel(client, messageBytes)
		}
	} else {
		log.Printf("WebSocket: User %d is offline, notification not delivered in real-time", userID)
	}
	h.mu.RUnlock()
}

// handleFullChannel handles the case when a client's send channel is full
func (h *Hub) handleFullChannel(client *Client, messageBytes []byte) {
	// Try to drain some messages from the channel
	select {
	case <-client.send:
		// Drained one message, try to send the notification again
		select {
		case client.send <- messageBytes:
			log.Printf("WebSocket: Notification sent to user %d after channel cleanup", client.userID)
		default:
			log.Printf("WebSocket: Failed to send notification to user %d - channel still full", client.userID)
			// Consider disconnecting the client if channel is consistently full
			h.unregisterClient(client)
		}
	case <-time.After(1 * time.Second):
		log.Printf("WebSocket: Timeout waiting for channel cleanup for user %d", client.userID)
		// Channel is consistently full, disconnect the client
		h.unregisterClient(client)
	}
}

// cleanupRoutine periodically cleans up inactive connections and updates presence
func (h *Hub) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.mu.RLock()
			var inactiveClients []*Client
			cutoff := time.Now().Add(-2 * time.Minute)

			for client := range h.clients {
				if client.lastSeen.Before(cutoff) {
					inactiveClients = append(inactiveClients, client)
				}
			}
			h.mu.RUnlock()

			// Remove inactive clients
			for _, client := range inactiveClients {
				log.Printf("WebSocket: Removing inactive client for user %d", client.userID)
				h.unregisterClient(client)
			}

		case <-h.stop:
			return
		}
	}
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID int, message WSMessage) {
	h.mu.RLock()
	client, exists := h.userClients[userID]
	h.mu.RUnlock()

	if exists {
		h.sendToClient(client, message)
	} else {
		// User is offline, could queue message for later delivery
		log.Printf("WebSocket: User %d is offline, message not delivered", userID)
	}
}

// BroadcastToGroup sends a message to all members of a group except the sender
func (h *Hub) BroadcastToGroup(groupID int, message WSMessage, senderID int) {
	h.mu.RLock()
	groupClients, exists := h.groupClients[groupID]
	h.mu.RUnlock()

	if !exists {
		log.Printf("WebSocket: No clients found for group %d", groupID)
		return
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("WebSocket: Error marshaling group message: %v", err)
		return
	}

	h.mu.RLock()
	for client := range groupClients {
		// Don't send to the sender
		if client.userID != senderID {
			select {
			case client.send <- messageBytes:
				log.Printf("WebSocket: Group message sent to user %d in group %d", client.userID, groupID)
			default:
				// Client's send channel is full, skip this client
				log.Printf("WebSocket: Client %d send channel full, skipping", client.userID)
			}
		}
	}
	h.mu.RUnlock()
}
