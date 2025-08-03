class App {
    constructor() {
        this.currentUser = null;
        this.wsManager = new WebSocketManager();
        this.chatManager = new ChatManager();
        this.isInitialized = false;
    }

    async initialize() {
        if (this.isInitialized) return;
        
        console.log('Initializing Ripple Messenger...');
        
        // Check authentication status
        const authStatus = await AuthManager.checkAuthStatus();
        
        if (authStatus.isAuthenticated) {
            this.currentUser = authStatus.user;
            await this.showChatScreen();
        } else {
            this.showLoginScreen();
        }
        
        this.setupGlobalEventListeners();
        this.isInitialized = true;
    }

    showLoginScreen() {
        document.getElementById('login-screen').classList.remove('hidden');
        document.getElementById('chat-screen').classList.add('hidden');
        
        const loginForm = document.getElementById('login-form');
        const emailInput = document.getElementById('email');
        const passwordInput = document.getElementById('password');
        const errorDiv = document.getElementById('login-error');
        const registerLink = document.getElementById('register-link');

        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const email = emailInput.value.trim();
            const password = passwordInput.value;
            
            if (!email || !password) {
                errorDiv.textContent = 'Please enter both email and password';
                return;
            }

            // Show loading state
            const submitBtn = loginForm.querySelector('button[type="submit"]');
            const originalText = submitBtn.textContent;
            submitBtn.textContent = 'Logging in...';
            submitBtn.disabled = true;
            errorDiv.textContent = '';

            try {
                const result = await AuthManager.login(email, password);
                
                if (result.success) {
                    this.currentUser = result.user;
                    await this.showChatScreen();
                } else {
                    errorDiv.textContent = result.error;
                }
            } catch (error) {
                errorDiv.textContent = 'An unexpected error occurred';
                console.error('Login error:', error);
            } finally {
                submitBtn.textContent = originalText;
                submitBtn.disabled = false;
            }
        });

        registerLink.addEventListener('click', (e) => {
            e.preventDefault();
            AuthManager.openRegistrationPage();
        });
    }

    async showChatScreen() {
        document.getElementById('login-screen').classList.add('hidden');
        document.getElementById('chat-screen').classList.remove('hidden');
        
        // Update user info in sidebar
        const userAvatar = document.getElementById('user-avatar');
        const userName = document.getElementById('user-name');
        const userStatus = document.getElementById('user-status');
        
        const avatarUrl = this.currentUser.avatar ? `http://localhost:8080${this.currentUser.avatar}` : 'default-avatar.svg';
        userAvatar.src = avatarUrl;
        userAvatar.onerror = () => { userAvatar.src = 'default-avatar.svg'; };
        userName.textContent = `${this.currentUser.first_name} ${this.currentUser.last_name}`;
        userStatus.classList.add('online');

        // Initialize chat functionality
        await this.chatManager.initialize();
        
        // Connect WebSocket
        const connected = await this.wsManager.connect();
        if (!connected) {
            console.warn('Failed to connect to WebSocket');
        }

        // Setup logout functionality
        const logoutBtn = document.getElementById('logout-btn');
        logoutBtn.addEventListener('click', () => {
            this.showLogoutMenu();
        });
    }

    showLogoutMenu() {
        // Create a simple context menu for logout
        const existingMenu = document.getElementById('logout-menu');
        if (existingMenu) {
            existingMenu.remove();
        }

        const menu = document.createElement('div');
        menu.id = 'logout-menu';
        menu.style.cssText = `
            position: absolute;
            top: 60px;
            right: 16px;
            background: white;
            border: 1px solid #ddd;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.1);
            z-index: 1000;
            min-width: 120px;
        `;

        menu.innerHTML = `
            <div style="padding: 8px 16px; cursor: pointer; border-bottom: 1px solid #f0f2f5;" onclick="app.showSettings()">
                Settings
            </div>
            <div style="padding: 8px 16px; cursor: pointer; color: #e74c3c;" onclick="app.logout()">
                Logout
            </div>
        `;

        document.body.appendChild(menu);

        // Close menu when clicking outside
        const closeMenu = (e) => {
            if (!menu.contains(e.target)) {
                menu.remove();
                document.removeEventListener('click', closeMenu);
            }
        };
        
        setTimeout(() => {
            document.addEventListener('click', closeMenu);
        }, 100);
    }

    showSettings() {
        // Simple settings dialog
        alert('Settings functionality would be implemented here');
        document.getElementById('logout-menu')?.remove();
    }

    async logout() {
        try {
            // Disconnect WebSocket
            this.wsManager.disconnect();
            
            // Clear authentication
            await AuthManager.logout();
            
            // Reset state
            this.currentUser = null;
            this.chatManager = new ChatManager();
            
            // Show login screen
            this.showLoginScreen();
            
            // Remove logout menu
            document.getElementById('logout-menu')?.remove();
            
        } catch (error) {
            console.error('Logout error:', error);
        }
    }

    setupGlobalEventListeners() {
        // Handle app focus for notifications
        window.addEventListener('focus', () => {
            // Clear badge count when app is focused
            window.electronAPI.setBadgeCount(0);
        });

        // Handle online/offline status
        window.addEventListener('online', () => {
            console.log('App is online');
            if (this.currentUser) {
                this.wsManager.connect();
            }
        });

        window.addEventListener('offline', () => {
            console.log('App is offline');
            this.wsManager.disconnect();
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + R to refresh
            if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
                e.preventDefault();
                location.reload();
            }
            
            // Escape to close modals/menus
            if (e.key === 'Escape') {
                document.getElementById('logout-menu')?.remove();
                document.getElementById('emoji-picker')?.classList.add('hidden');
            }
        });
    }
}

// Initialize app when DOM is loaded
let app;
document.addEventListener('DOMContentLoaded', () => {
    app = new App();
    app.initialize().catch(error => {
        console.error('Failed to initialize app:', error);
    });
});