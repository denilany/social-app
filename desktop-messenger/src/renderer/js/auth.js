class AuthManager {
    static async login(email, password) {
        try {
            console.log('AuthManager: Attempting login for email:', email);
            const fetchOptions = await Config.getFetchOptions();
            const response = await fetch(Config.ENDPOINTS.AUTH.LOGIN, {
                method: 'POST',
                ...fetchOptions,
                body: JSON.stringify({ email, password })
            });

            console.log('AuthManager: Login response status:', response.status);
            const data = await response.json();
            console.log('AuthManager: Login response data:', data);

            if (response.ok) {
                // Store user data locally for offline access
                await Storage.setUserData(data.data.user);

                // Store session token if provided
                if (data.data.session_token) {
                    await Storage.setSessionToken(data.data.session_token);
                    console.log('AuthManager: Session token stored');
                }

                console.log(data.data.user)
                console.log('AuthManager: Login successful, user data stored');
                return { success: true, user: data.data.user };
            } else {
                const errorMessage = data.error?.message || data.message || 'Login failed';
                console.error('AuthManager: Login failed:', errorMessage);
                return { success: false, error: errorMessage };
            }
        } catch (error) {
            console.error('AuthManager: Login network error:', error);
            return { success: false, error: `Network error. Please check your connection: ${error.message}` };
        }
    }

    static async logout() {
        try {
            console.log('AuthManager: Attempting logout');
            const fetchOptions = await Config.getFetchOptions();
            await fetch(Config.ENDPOINTS.AUTH.LOGOUT, {
                method: 'POST',
                ...fetchOptions
            });
            console.log('AuthManager: Logout request completed');
        } catch (error) {
            console.error('AuthManager: Logout network error:', error);
        } finally {
            await Storage.clearAuth();
            console.log('AuthManager: Local auth data cleared');
        }
    }

    static async checkAuthStatus() {
        try {
            console.log('AuthManager: Checking authentication status');
            const fetchOptions = await Config.getFetchOptions();
            const response = await fetch(Config.ENDPOINTS.AUTH.PROFILE, fetchOptions);

            console.log('AuthManager: Auth check response status:', response.status);

            if (response.ok) {
                const data = await response.json();
                console.log('AuthManager: Auth check successful, user authenticated');
                // Update stored user data
                await Storage.setUserData(data.data);
                return { isAuthenticated: true, user: data.data };
            } else {
                console.log('AuthManager: Auth check failed, clearing stored auth data');
                await Storage.clearAuth();
                return { isAuthenticated: false };
            }
        } catch (error) {
            console.error('AuthManager: Auth check network error:', error);
            // Try to use cached user data if offline
            const userData = await Storage.getUserData();
            if (userData) {
                console.log('AuthManager: Using cached user data (offline mode)');
                return { isAuthenticated: true, user: userData };
            }
            console.log('AuthManager: No cached user data available');
            return { isAuthenticated: false };
        }
    }

    static async getContacts() {
        try {
            console.log('AuthManager: Fetching contacts');
            const fetchOptions = await Config.getFetchOptions();
            const response = await fetch(Config.ENDPOINTS.CHAT.FOLLOWED_USERS, fetchOptions);

            console.log('AuthManager: Contacts response status:', response.status);

            if (response.ok) {
                const data = await response.json();
                console.log('AuthManager: Contacts fetched successfully, count:', data.users?.length || 0);
                return data.users || [];
            } else {
                console.error('AuthManager: Failed to fetch contacts, status:', response.status);
            }
        } catch (error) {
            console.error('AuthManager: Error fetching contacts:', error);
        }

        return [];
    }

    static async getMessageHistory(otherUserId, limit = 50, offset = 0) {
        try {
            const fetchOptions = await Config.getFetchOptions();
            const response = await fetch(`${Config.ENDPOINTS.CHAT.MESSAGES_PRIVATE(otherUserId)}?limit=${limit}&offset=${offset}`, fetchOptions);

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
            const fetchOptions = await Config.getFetchOptions();
            const response = await fetch(Config.ENDPOINTS.CHAT.ONLINE_USERS, fetchOptions);

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
        if (window.electronAPI && window.electronAPI.openExternalUrl) {
            // Use Electron's shell.openExternal to open in system browser
            window.electronAPI.openExternalUrl('http://localhost:3000/register');
        } else {
            // Fallback for non-Electron environments
            window.open('http://localhost:3000/register', '_blank');
        }
    }
}