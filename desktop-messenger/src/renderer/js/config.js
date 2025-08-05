// Configuration for the Electron desktop messenger
class Config {

    static get WS_BASE_URL() {
        // Convert HTTP URL to WebSocket URL
        return this.API_BASE_URL.replace('http://', 'ws://').replace('https://', 'wss://');
    }

    static get CLIENT_IDENTIFIER() {
        return 'electron-desktop';
    }

    static get ENDPOINTS() {
        return {
            AUTH: {
                LOGIN: `${this.API_BASE_URL}/api/auth/login`,
                LOGOUT: `${this.API_BASE_URL}/api/auth/logout`,
                PROFILE: `${this.API_BASE_URL}/api/auth/profile`,
            },
            CHAT: {
                FOLLOWED_USERS: `${this.API_BASE_URL}/api/chat/followed-users`,
                MESSAGES_PRIVATE: (userId) => `${this.API_BASE_URL}/api/chat/messages/private/${userId}`,
                ONLINE_USERS: `${this.API_BASE_URL}/api/chat/online`,
            },
            WEBSOCKET: `${this.WS_BASE_URL}/ws`
        };
    }

    static getDefaultHeaders() {
        return {
            'Content-Type': 'application/json',
            'x-client': this.CLIENT_IDENTIFIER,
        };
    }

    static getFetchOptions(additionalOptions = {}) {
        return {
            credentials: 'include',
            headers: this.getDefaultHeaders(),
            ...additionalOptions
        };
    }

    // Method to override the API base URL if needed
    static setApiBaseUrl(url) {
        this._customApiBaseUrl = url;
    }

    // Updated API_BASE_URL getter to check for custom URL
    static get API_BASE_URL() {
        return this._customApiBaseUrl || 'http://localhost:8000';
    }
}

// Make Config available globally
window.Config = Config;
