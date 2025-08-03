class ChatManager {
    constructor() {
        this.currentChatUser = null;
        this.contacts = [];
        this.onlineUsers = new Set();
        this.typingUsers = new Map();
        this.unreadCounts = new Map();
        this.searchTerm = '';
    }

    async initialize() {
        await this.loadContacts();
        this.setupEventListeners();
        this.setupWebSocketHandlers();
    }

    async loadContacts() {
        this.contacts = await AuthManager.getContacts();
        
        // Get online users and update status
        const onlineUsers = await AuthManager.getOnlineUsers();
        this.onlineUsers.clear();
        onlineUsers.forEach(user => {
            this.onlineUsers.add(user.id);
        });
        
        this.renderContacts();
    }

    setupEventListeners() {
        // Search functionality
        const searchInput = document.getElementById('search-input');
        searchInput.addEventListener('input', (e) => {
            this.searchTerm = e.target.value.toLowerCase();
            this.renderContacts();
        });

        // Message input
        const messageInput = document.getElementById('message-input');
        const sendBtn = document.getElementById('send-btn');

        messageInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendMessage();
            }
        });

        sendBtn.addEventListener('click', () => this.sendMessage());

        // Typing indicator
        let typingTimer;
        messageInput.addEventListener('input', () => {
            if (this.currentChatUser) {
                app.wsManager.sendTyping(this.currentChatUser.id, true);
                clearTimeout(typingTimer);
                typingTimer = setTimeout(() => {
                    app.wsManager.sendTyping(this.currentChatUser.id, false);
                }, 1000);
            }
        });

        // Emoji picker
        const emojiBtn = document.getElementById('emoji-btn');
        const emojiPicker = document.getElementById('emoji-picker');
        
        emojiBtn.addEventListener('click', () => {
            emojiPicker.classList.toggle('hidden');
        });

        document.addEventListener('click', (e) => {
            if (!emojiBtn.contains(e.target) && !emojiPicker.contains(e.target)) {
                emojiPicker.classList.add('hidden');
            }
        });

        // Emoji selection
        emojiPicker.addEventListener('click', (e) => {
            if (e.target.classList.contains('emoji')) {
                messageInput.value += e.target.textContent;
                messageInput.focus();
            }
        });
    }

    setupWebSocketHandlers() {
        app.wsManager.on('message', (data) => {
            this.handleIncomingMessage(data);
        });

        app.wsManager.on('userStatus', (data) => {
            this.handleUserStatusUpdate(data);
        });

        app.wsManager.on('typing', (data) => {
            this.handleTypingIndicator(data);
        });

        app.wsManager.on('notification', (data) => {
            this.showNotification(data);
        });
        
        app.wsManager.on('userList', (data) => {
            if (data.online_users) {
                this.onlineUsers.clear();
                data.online_users.forEach(userId => {
                    this.onlineUsers.add(userId);
                });
                this.renderContacts();
            }
        });
    }

    renderContacts() {
        const contactsList = document.getElementById('contacts-list');
        const filteredContacts = this.contacts.filter(contact => 
            contact.first_name.toLowerCase().includes(this.searchTerm) ||
            contact.last_name.toLowerCase().includes(this.searchTerm)
        );

        contactsList.innerHTML = filteredContacts.map(contact => {
            const isOnline = this.onlineUsers.has(contact.id);
            const unreadCount = this.unreadCounts.get(contact.id) || 0;
            const isActive = this.currentChatUser && this.currentChatUser.id === contact.id;
            const avatarUrl = contact.avatar ? `http://localhost:8080/uploads/${contact.avatar}` : 'default-avatar.svg';

            return `
                <div class="contact-item ${isActive ? 'active' : ''}" data-user-id="${contact.id}">
                    <img src="${avatarUrl}" alt="Avatar" class="avatar" onerror="this.src='default-avatar.svg'">
                    <div class="contact-info">
                        <div class="contact-name">${contact.first_name} ${contact.last_name}</div>
                        <div class="contact-status">${isOnline ? 'Online' : 'Offline'}</div>
                    </div>
                    ${unreadCount > 0 ? `<div class="unread-count">${unreadCount}</div>` : ''}
                </div>
            `;
        }).join('');

        // Add click handlers
        contactsList.querySelectorAll('.contact-item').forEach(item => {
            item.addEventListener('click', () => {
                const userId = parseInt(item.dataset.userId);
                const user = this.contacts.find(c => c.id === userId);
                this.openChat(user);
            });
        });

        this.updateOnlineCount();
    }

    async openChat(user) {
        this.currentChatUser = user;
        
        // Update UI
        document.getElementById('chat-header').classList.remove('hidden');
        document.getElementById('message-input-container').classList.remove('hidden');
        
        const chatAvatar = document.getElementById('chat-avatar');
        const chatName = document.getElementById('chat-name');
        const chatStatus = document.getElementById('chat-status');
        
        const avatarUrl = user.avatar ? `http://localhost:8080/uploads/${user.avatar}` : 'default-avatar.svg';
        chatAvatar.src = avatarUrl;
        chatAvatar.onerror = () => { chatAvatar.src = 'default-avatar.svg'; };
        chatName.textContent = `${user.first_name} ${user.last_name}`;
        chatStatus.textContent = this.onlineUsers.has(user.id) ? 'Online' : 'Offline';

        // Clear unread count
        this.unreadCounts.delete(user.id);
        this.renderContacts();

        // Load and display messages
        await this.loadMessages(user.id);
    }

    async loadMessages(userId) {
        // First load from local storage for offline viewing
        const localMessages = await Storage.getMessages(userId);
        
        // Try to fetch recent messages from server
        try {
            const serverMessages = await AuthManager.getMessageHistory(userId, 50, 0);
            if (serverMessages.length > 0) {
                // Store server messages locally
                await Storage.setMessages(userId, serverMessages);
                this.displayMessages(serverMessages);
            } else {
                // Fall back to local messages
                this.displayMessages(localMessages);
            }
        } catch (error) {
            console.error('Failed to load messages from server:', error);
            // Fall back to local messages
            this.displayMessages(localMessages);
        }
    }
    
    displayMessages(messages) {
        const messagesContainer = document.getElementById('messages-container');
        messagesContainer.innerHTML = messages.map(msg => this.createMessageElement(msg)).join('');
        this.scrollToBottom();
    }

    createMessageElement(message) {
        const isSent = message.sender_id === app.currentUser.id;
        const timestamp = message.timestamp || message.created_at;
        const time = new Date(timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
        
        return `
            <div class="message ${isSent ? 'sent' : 'received'}">
                <div class="message-content">${this.escapeHtml(message.content)}</div>
                <div class="message-time">${time}</div>
            </div>
        `;
    }

    async sendMessage() {
        const messageInput = document.getElementById('message-input');
        const content = messageInput.value.trim();
        
        if (!content || !this.currentChatUser) return;

        const message = {
            id: Date.now(),
            sender_id: app.currentUser.id,
            receiver_id: this.currentChatUser.id,
            content: content,
            timestamp: new Date().toISOString()
        };

        // Add to local storage
        await Storage.addMessage(this.currentChatUser.id, message);
        
        // Display message immediately for better UX
        this.displayMessage(message);
        
        // Send via WebSocket
        app.wsManager.sendMessage(this.currentChatUser.id, content);
        
        messageInput.value = '';
        messageInput.focus();
    }

    async handleIncomingMessage(data) {
        const message = {
            id: data.id || Date.now(),
            sender_id: data.sender_id,
            receiver_id: data.recipient_id,
            content: data.content,
            timestamp: data.timestamp,
            created_at: data.timestamp
        };

        // Store message
        await Storage.addMessage(data.sender_id, message);

        // If chat is open with sender, display message
        if (this.currentChatUser && this.currentChatUser.id === data.sender_id) {
            this.displayMessage(message);
        } else {
            // Update unread count
            const currentCount = this.unreadCounts.get(data.sender_id) || 0;
            this.unreadCounts.set(data.sender_id, currentCount + 1);
            this.renderContacts();
        }

        // Show notification
        const sender = this.contacts.find(c => c.id === data.sender_id);
        if (sender) {
            const avatarUrl = sender.avatar ? `http://localhost:8080/uploads/${sender.avatar}` : null;
            this.showNotification({
                title: `${sender.first_name} ${sender.last_name}`,
                body: data.content,
                icon: avatarUrl
            });
        }
    }

    displayMessage(message) {
        const messagesContainer = document.getElementById('messages-container');
        const messageElement = document.createElement('div');
        messageElement.innerHTML = this.createMessageElement(message);
        messagesContainer.appendChild(messageElement.firstElementChild);
        this.scrollToBottom();
    }

    handleUserStatusUpdate(data) {
        if (data.is_online) {
            this.onlineUsers.add(data.user_id);
        } else {
            this.onlineUsers.delete(data.user_id);
        }
        
        this.renderContacts();
        
        // Update current chat status
        if (this.currentChatUser && this.currentChatUser.id === data.user_id) {
            const chatStatus = document.getElementById('chat-status');
            chatStatus.textContent = data.is_online ? 'Online' : 'Offline';
        }
    }

    handleTypingIndicator(data) {
        if (this.currentChatUser && this.currentChatUser.id === data.sender_id) {
            const typingIndicator = document.getElementById('typing-indicator');
            
            if (data.is_typing) {
                const sender = this.contacts.find(c => c.id === data.sender_id);
                typingIndicator.textContent = `${sender.first_name} is typing...`;
                typingIndicator.style.display = 'block';
            } else {
                typingIndicator.style.display = 'none';
            }
        }
    }

    updateOnlineCount() {
        const onlineCount = document.getElementById('online-count');
        onlineCount.textContent = this.onlineUsers.size;
    }

    scrollToBottom() {
        const messagesContainer = document.getElementById('messages-container');
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    showNotification(data) {
        window.electronAPI.showNotification({
            title: data.title,
            body: data.body,
            icon: data.icon
        });
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}