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

    static async getDefaultHeaders() {
        const headers = {
            'Content-Type': 'application/json',
            'x-client': this.CLIENT_IDENTIFIER,
        };

        // Add session token if available
        try {
            const sessionToken = await Storage.getSessionToken();
            if (sessionToken) {
                headers['Authorization'] = `Bearer ${sessionToken}`;
            }
        } catch (error) {
            console.warn('Config: Could not retrieve session token:', error);
        }

        return headers;
    }

    static async getFetchOptions(additionalOptions = {}) {
        const defaultHeaders = await this.getDefaultHeaders();
        return {
            credentials: 'include',
            headers: {
                ...defaultHeaders,
                ...(additionalOptions.headers || {})
            },
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
