'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Sidebar from '@/components/Sidebar';
import api from '@/lib/api';

export default function SettingsPage() {
    const router = useRouter();
    const [user, setUser] = useState(null);
    const [displayName, setDisplayName] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [saving, setSaving] = useState(false);
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');

    useEffect(() => {
        fetchProfile();
    }, []);

    const fetchProfile = async () => {
        try {
            const data = await api.getProfile();
            setUser(data);
            setDisplayName(data.display_name || '');
        } catch {
            const cached = api.getUser();
            if (cached) {
                setUser(cached);
                setDisplayName(cached.display_name || '');
            }
        }
    };

    const handleSaveProfile = async (e) => {
        e.preventDefault();
        setError('');
        setMessage('');
        setSaving(true);

        const updates = {};
        if (displayName !== (user?.display_name || '')) {
            updates.display_name = displayName;
        }

        if (password) {
            if (password !== confirmPassword) {
                setError('Passwords do not match');
                setSaving(false);
                return;
            }
            if (password.length < 8) {
                setError('Password must be at least 8 characters');
                setSaving(false);
                return;
            }
            updates.password = password;
        }

        if (Object.keys(updates).length === 0) {
            setMessage('No changes to save');
            setSaving(false);
            return;
        }

        try {
            const updated = await api.updateProfile(updates);
            setUser(updated);
            api.setUser(updated);
            setPassword('');
            setConfirmPassword('');
            setMessage('Profile updated successfully');
        } catch (err) {
            setError(err.message);
        } finally {
            setSaving(false);
        }
    };

    const handleLogout = () => {
        api.logout();
        router.push('/login');
    };

    return (
        <div className="app-layout">
            <Sidebar />
            <main className="main-content">
                <div className="page-header">
                    <h2>Settings</h2>
                </div>

                <div className="glass-card" style={{ maxWidth: 600 }}>
                    {/* Account Info */}
                    <div className="settings-section">
                        <h3>Account Information</h3>
                        {user && (
                            <div style={{ display: 'grid', gridTemplateColumns: '120px 1fr', gap: '10px 16px', fontSize: 14 }}>
                                <span style={{ color: 'var(--text-muted)' }}>Email</span>
                                <span>{user.email}</span>
                                <span style={{ color: 'var(--text-muted)' }}>Domain</span>
                                <span>{user.domain_name || '—'}</span>
                                <span style={{ color: 'var(--text-muted)' }}>Role</span>
                                <span>{user.is_admin ? '👑 Admin' : 'User'}</span>
                                <span style={{ color: 'var(--text-muted)' }}>Storage</span>
                                <span>
                                    {user.storage_used_mb} / {user.storage_quota_mb} MB
                                    <div style={{
                                        width: '100%',
                                        height: 4,
                                        background: 'var(--bg-primary)',
                                        borderRadius: 2,
                                        marginTop: 4,
                                        overflow: 'hidden',
                                    }}>
                                        <div style={{
                                            width: `${Math.min((user.storage_used_mb / user.storage_quota_mb) * 100, 100)}%`,
                                            height: '100%',
                                            background: 'linear-gradient(90deg, var(--accent), #a855f7)',
                                            borderRadius: 2,
                                        }} />
                                    </div>
                                </span>
                                <span style={{ color: 'var(--text-muted)' }}>Member since</span>
                                <span>{new Date(user.created_at).toLocaleDateString()}</span>
                            </div>
                        )}
                    </div>

                    {/* Profile Edit */}
                    <div className="settings-section">
                        <h3>Edit Profile</h3>

                        {message && (
                            <div className="toast success" style={{ position: 'relative', bottom: 'auto', right: 'auto', marginBottom: 12 }}>
                                ✓ {message}
                            </div>
                        )}
                        {error && <div className="auth-error">{error}</div>}

                        <form onSubmit={handleSaveProfile}>
                            <div className="form-group">
                                <label className="form-label" htmlFor="settings-name">Display Name</label>
                                <input
                                    id="settings-name"
                                    className="form-input"
                                    type="text"
                                    value={displayName}
                                    onChange={(e) => setDisplayName(e.target.value)}
                                />
                            </div>

                            <div className="form-group">
                                <label className="form-label" htmlFor="settings-pwd">New Password</label>
                                <input
                                    id="settings-pwd"
                                    className="form-input"
                                    type="password"
                                    placeholder="Leave blank to keep current"
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                />
                            </div>

                            {password && (
                                <div className="form-group">
                                    <label className="form-label" htmlFor="settings-confirm">Confirm Password</label>
                                    <input
                                        id="settings-confirm"
                                        className="form-input"
                                        type="password"
                                        placeholder="Confirm new password"
                                        value={confirmPassword}
                                        onChange={(e) => setConfirmPassword(e.target.value)}
                                    />
                                </div>
                            )}

                            <button className="btn btn-primary" type="submit" disabled={saving}>
                                {saving ? 'Saving...' : 'Save Changes'}
                            </button>
                        </form>
                    </div>

                    {/* Danger Zone */}
                    <div className="settings-section">
                        <h3>Session</h3>
                        <button className="btn btn-danger" onClick={handleLogout}>
                            🚪 Sign Out
                        </button>
                    </div>
                </div>
            </main>
        </div>
    );
}
