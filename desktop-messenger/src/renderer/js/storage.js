class Storage {
    static async get(key) {
        return await window.electronAPI.getStoredData(key);
    }

    static async set(key, value) {
        return await window.electronAPI.setStoredData(key, value);
    }

    static async delete(key) {
        return await window.electronAPI.deleteStoredData(key);
    }



    static async getUserData() {
        return await this.get('userData');
    }

    static async setUserData(userData) {
        return await this.set('userData', userData);
    }

    static async getSessionToken() {
        return await this.get('sessionToken');
    }

    static async setSessionToken(token) {
        return await this.set('sessionToken', token);
    }

    static async getMessages(userId) {
        const messages = await this.get('messages') || {};
        return messages[userId] || [];
    }

    static async setMessages(userId, messages) {
        const allMessages = await this.get('messages') || {};
        allMessages[userId] = messages;
        return await this.set('messages', allMessages);
    }

    static async addMessage(userId, message) {
        const messages = await this.getMessages(userId);
        messages.push(message);
        return await this.setMessages(userId, messages);
    }

    static async clearAuth() {
        await this.delete('userData');
        await this.delete('sessionToken');
    }
}