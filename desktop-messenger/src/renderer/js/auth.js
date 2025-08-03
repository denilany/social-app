class AuthManager {
    static async login(email, password) {
        try {
            const response = await fetch('http://localhost:8000/api/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ email, password }),
                credentials: 'include'
            });

            const data = await response.json();

            if (response.ok) {
                // Store user data locally for offline access
                await Storage.setUserData(data.data.user);
                return { success: true, user: data.data.user };
            } else {
                return { success: false, error: data.error?.message || data.message || 'Login failed' };
            }
        } catch (error) {
            console.error('Login error:', error);
            return { success: false, error: `Network error. Please check your connection: ${error}` };
        }
    }

    static async logout() {
        try {
            await fetch('http://localhost:8000/api/auth/logout', {
                method: 'POST',
                credentials: 'include'
            });
        } catch (error) {
            console.error('Logout error:', error);
        } finally {
            await Storage.clearAuth();
        }
    }

    static async checkAuthStatus() {
        try {
            const response = await fetch('http://localhost:8000/api/auth/profile', {
                credentials: 'include'
            });

            if (response.ok) {
                const data = await response.json();
                // Update stored user data
                await Storage.setUserData(data.data);
                return { isAuthenticated: true, user: data.data };
            } else {
                await Storage.clearAuth();
                return { isAuthenticated: false };
            }
        } catch (error) {
            console.error('Auth check error:', error);
            // Try to use cached user data if offline
            const userData = await Storage.getUserData();
            if (userData) {
                return { isAuthenticated: true, user: userData };
            }
            return { isAuthenticated: false };
        }
    }

    static async getContacts() {
        try {
            const response = await fetch('http://localhost:8000/api/chat/followed-users', {
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
        try {
            const response = await fetch(`http://localhost:8000/api/chat/messages/private/${otherUserId}?limit=${limit}&offset=${offset}`, {
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
        try {
            const response = await fetch('http://localhost:8000/api/chat/online', {
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