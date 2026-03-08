/**
 * smail API Client
 * Centralized fetch wrapper with JWT auth.
 */

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000';

class ApiClient {
    getToken() {
        if (typeof window !== 'undefined') {
            return localStorage.getItem('smail_token');
        }
        return null;
    }

    setToken(token) {
        localStorage.setItem('smail_token', token);
    }

    clearToken() {
        localStorage.removeItem('smail_token');
        localStorage.removeItem('smail_user');
    }

    getUser() {
        if (typeof window !== 'undefined') {
            const user = localStorage.getItem('smail_user');
            return user ? JSON.parse(user) : null;
        }
        return null;
    }

    setUser(user) {
        localStorage.setItem('smail_user', JSON.stringify(user));
    }

    async request(endpoint, options = {}) {
        const url = `${API_URL}${endpoint}`;
        const token = this.getToken();

        const headers = {
            'Content-Type': 'application/json',
            ...options.headers,
        };

        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        const response = await fetch(url, {
            ...options,
            headers,
        });

        if (response.status === 401) {
            this.clearToken();
            if (typeof window !== 'undefined') {
                window.location.href = '/login';
            }
            throw new Error('Unauthorized');
        }

        if (!response.ok) {
            const error = await response.json().catch(() => ({ detail: 'An error occurred' }));
            throw new Error(error.detail || `HTTP ${response.status}`);
        }

        return response.json();
    }

    // Auth
    async login(email, password) {
        const data = await this.request('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
        this.setToken(data.access_token);
        this.setUser(data.user);
        return data;
    }

    async register(email, password, displayName) {
        const data = await this.request('/auth/register', {
            method: 'POST',
            body: JSON.stringify({ email, password, display_name: displayName }),
        });
        this.setToken(data.access_token);
        this.setUser(data.user);
        return data;
    }

    logout() {
        this.clearToken();
    }

    // Mail
    async getInbox(folder = 'INBOX', page = 1, perPage = 50) {
        return this.request(`/mail/inbox?folder=${encodeURIComponent(folder)}&page=${page}&per_page=${perPage}`);
    }

    async getMail(id) {
        return this.request(`/mail/${id}`);
    }

    async sendMail(to, subject, body, htmlBody = null) {
        return this.request('/mail/send', {
            method: 'POST',
            body: JSON.stringify({ to, subject, body, html_body: htmlBody }),
        });
    }

    async mailAction(id, action, folder = null) {
        return this.request(`/mail/${id}/action`, {
            method: 'POST',
            body: JSON.stringify({ action, folder }),
        });
    }

    async getFolders() {
        return this.request('/mail/folders');
    }

    // Domains
    async getDomains() {
        return this.request('/domains');
    }

    async createDomain(domain) {
        return this.request('/domains', {
            method: 'POST',
            body: JSON.stringify({ domain }),
        });
    }

    async getDomainDns(domainId) {
        return this.request(`/domains/${domainId}/dns`);
    }

    async deleteDomain(domainId) {
        return this.request(`/domains/${domainId}`, { method: 'DELETE' });
    }

    // Users
    async getUsers() {
        return this.request('/users');
    }

    async createUser(userData) {
        return this.request('/users', {
            method: 'POST',
            body: JSON.stringify(userData),
        });
    }

    async getProfile() {
        return this.request('/users/me');
    }

    async updateProfile(data) {
        return this.request('/users/me', {
            method: 'PATCH',
            body: JSON.stringify(data),
        });
    }
}

const api = new ApiClient();
export default api;
