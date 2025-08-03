class AuthManager {
    static async login(email, password) {
        try {
            const response = await fetch('http://localhost:8080/api/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ email, password }),
                credentials: 'include'
            });

            const data = await response.json();

            if (response.ok) {
                await Storage.setAuthToken(data.session_token);
                await Storage.setUserData(data.user);
                return { success: true, user: data.user };
            } else {
                return { success: false, error: data.error || 'Login failed' };
            }
        } catch (error) {
            console.error('Login error:', error);
            return { success: false, error: `Network error. Please check your connection: ${error}` };
        }
    }

    static async logout() {
        try {
            const token = await Storage.getAuthToken();
            if (token) {
                await fetch('http://localhost:8080/api/auth/logout', {
                    method: 'POST',
                    credentials: 'include'
                });
            }
        } catch (error) {
            console.error('Logout error:', error);
        } finally {
            await Storage.clearAuth();
        }
    }

    static async checkAuthStatus() {
        const token = await Storage.getAuthToken();
        const userData = await Storage.getUserData();

        if (!token || !userData) {
            return { isAuthenticated: false };
        }

        try {
            const response = await fetch('http://localhost:8080/api/auth/profile', {
                credentials: 'include'
            });

            if (response.ok) {
                return { isAuthenticated: true, user: userData };
            } else {
                await Storage.clearAuth();
                return { isAuthenticated: false };
            }
        } catch (error) {
            console.error('Auth check error:', error);
            return { isAuthenticated: true, user: userData }; // Assume valid if offline
        }
    }

    static async getContacts() {
        const token = await Storage.getAuthToken();
        if (!token) return [];

        try {
            // Set session cookie for API requests
            document.cookie = `session_id=${token}; path=/`;
            
            const response = await fetch('http://localhost:8080/api/chat/followed-users', {
                credentials: 'include'
            });

            if (response.ok) {
                const data = await response.json();
                return data.users || [];
            }
        } catch (error) {
            console.error('Error fetching contacts:', error);
        }

        return [];
    }

    static async getMessageHistory(otherUserId, limit = 50, offset = 0) {
        const token = await Storage.getAuthToken();
        if (!token) return [];

        try {
            document.cookie = `session_id=${token}; path=/`;
            
            const response = await fetch(`http://localhost:8080/api/chat/messages/private/${otherUserId}?limit=${limit}&offset=${offset}`, {
                credentials: 'include'
            });

            if (response.ok) {
                const data = await response.json();
                return data.messages || [];
            }
        } catch (error) {
            console.error('Error fetching message history:', error);
        }

        return [];
    }

    static async getOnlineUsers() {
        const token = await Storage.getAuthToken();
        if (!token) return [];

        try {
            document.cookie = `session_id=${token}; path=/`;
            
            const response = await fetch('http://localhost:8080/api/chat/online', {
                credentials: 'include'
            });

            if (response.ok) {
                const data = await response.json();
                return data.online_users || [];
            }
        } catch (error) {
            console.error('Error fetching online users:', error);
        }

        return [];
    }

    static openRegistrationPage() {
        // Open the web registration page in the default browser
        if (window.electronAPI) {
            // We're in Electron, but can't access shell directly from renderer
            // Instead, we'll open in a new window
            window.open('http://localhost:3000/register', '_blank');
        } else {
            window.open('http://localhost:3000/register', '_blank');
        }
    }
}