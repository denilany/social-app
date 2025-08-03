class WebSocketManager {
    constructor() {
        this.ws = null;
        this.isConnected = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.messageQueue = [];
        this.eventHandlers = {};
    }

    async connect() {
        const token = await Storage.getAuthToken();
        if (!token) return false;

        try {
            // Set session cookie for WebSocket authentication
            document.cookie = `session_id=${token}; path=/`;
            
            this.ws = new WebSocket('ws://localhost:8000/ws');
            
            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.isConnected = true;
                this.reconnectAttempts = 0;
                this.updateConnectionStatus(true);
                this.flushMessageQueue();
                this.emit('connected');
            };

            this.ws.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    this.handleMessage(data);
                } catch (error) {
                    console.error('Error parsing WebSocket message:', error);
                }
            };

            this.ws.onclose = () => {
                console.log('WebSocket disconnected');
                this.isConnected = false;
                this.updateConnectionStatus(false);
                this.emit('disconnected');
                this.attemptReconnect();
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.emit('error', error);
            };

            return true;
        } catch (error) {
            console.error('Failed to connect WebSocket:', error);
            return false;
        }
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.isConnected = false;
        this.updateConnectionStatus(false);
    }

    send(data) {
        if (this.isConnected && this.ws) {
            this.ws.send(JSON.stringify(data));
        } else {
            this.messageQueue.push(data);
        }
    }

    flushMessageQueue() {
        while (this.messageQueue.length > 0) {
            const message = this.messageQueue.shift();
            this.send(message);
        }
    }

    attemptReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            console.log(`Attempting to reconnect... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
            
            setTimeout(() => {
                this.connect();
            }, this.reconnectDelay * this.reconnectAttempts);
        }
    }

    handleMessage(data) {
        switch (data.type) {
            case 'private_message':
                this.emit('message', {
                    id: data.message_id,
                    sender_id: data.from,
                    recipient_id: data.to,
                    content: data.content,
                    timestamp: data.timestamp
                });
                break;
            case 'user_online':
                this.emit('userStatus', {
                    user_id: data.data?.user_id || data.from,
                    is_online: true
                });
                break;
            case 'user_offline':
                this.emit('userStatus', {
                    user_id: data.data?.user_id || data.from,
                    is_online: false
                });
                break;
            case 'user_list':
                this.emit('userList', data.data);
                break;
            case 'typing':
                this.emit('typing', {
                    sender_id: data.from,
                    is_typing: data.data?.is_typing
                });
                break;
            case 'notification':
                this.emit('notification', data.data);
                break;
            case 'connection_status':
                console.log('Connection status:', data.data);
                break;
            case 'ping':
                this.send({ type: 'pong', timestamp: new Date().toISOString() });
                break;
            default:
                console.log('Unknown message type:', data.type);
        }
    }

    updateConnectionStatus(isOnline) {
        const offlineIndicator = document.getElementById('offline-indicator');
        const sendBtn = document.getElementById('send-btn');
        const messageInput = document.getElementById('message-input');

        if (isOnline) {
            offlineIndicator.classList.add('hidden');
            if (sendBtn) sendBtn.disabled = false;
            if (messageInput) messageInput.disabled = false;
        } else {
            offlineIndicator.classList.remove('hidden');
            if (sendBtn) sendBtn.disabled = true;
            if (messageInput) messageInput.disabled = true;
        }
    }

    on(event, handler) {
        if (!this.eventHandlers[event]) {
            this.eventHandlers[event] = [];
        }
        this.eventHandlers[event].push(handler);
    }

    emit(event, data) {
        if (this.eventHandlers[event]) {
            this.eventHandlers[event].forEach(handler => handler(data));
        }
    }

    sendMessage(recipientId, content) {
        this.send({
            type: 'private_message',
            to: recipientId,
            content: content,
            timestamp: new Date().toISOString()
        });
    }

    sendTyping(recipientId, isTyping) {
        this.send({
            type: 'typing',
            data: {
                chat_type: 'private',
                target_id: recipientId,
                is_typing: isTyping
            },
            timestamp: new Date().toISOString()
        });
    }
}