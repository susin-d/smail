'use client';

import { usePathname, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import api from '@/lib/api';

export default function Sidebar() {
    const pathname = usePathname();
    const router = useRouter();
    const [user, setUser] = useState(null);
    const [folders, setFolders] = useState([]);

    useEffect(() => {
        const u = api.getUser();
        setUser(u);

        api.getFolders().then(setFolders).catch(() => {
            setFolders([
                { name: 'INBOX', count: 0, unread: 0 },
                { name: 'Sent', count: 0, unread: 0 },
                { name: 'Drafts', count: 0, unread: 0 },
                { name: 'Trash', count: 0, unread: 0 },
                { name: 'Spam', count: 0, unread: 0 },
            ]);
        });
    }, []);

    const folderIcons = {
        'INBOX': '📥',
        'Sent': '📤',
        'Drafts': '📝',
        'Trash': '🗑️',
        'Spam': '⚠️',
        'Archive': '📦',
    };

    const navigate = (path) => router.push(path);

    const handleLogout = () => {
        api.logout();
        router.push('/login');
    };

    const initials = user?.display_name
        ? user.display_name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
        : user?.email?.slice(0, 2).toUpperCase() || '?';

    return (
        <aside className="sidebar">
            {/* Logo */}
            <div className="sidebar-logo">
                <div className="logo-icon">✉</div>
                <h1>MaaS</h1>
            </div>

            {/* Compose Button */}
            <div style={{ padding: '0 12px', marginBottom: '8px' }}>
                <button
                    className="btn btn-primary"
                    style={{ width: '100%' }}
                    onClick={() => navigate('/compose')}
                >
                    ✏️ Compose
                </button>
            </div>

            {/* Navigation */}
            <nav className="sidebar-nav">
                <div className="sidebar-section-title">Folders</div>
                {folders.map((folder) => (
                    <div
                        key={folder.name}
                        className={`nav-item ${pathname === '/inbox' && folder.name === 'INBOX' ? 'active' : ''}`}
                        onClick={() => navigate(`/inbox?folder=${encodeURIComponent(folder.name)}`)}
                    >
                        <span className="icon">{folderIcons[folder.name] || '📁'}</span>
                        <span>{folder.name === 'INBOX' ? 'Inbox' : folder.name}</span>
                        {folder.unread > 0 && <span className="badge">{folder.unread}</span>}
                    </div>
                ))}

                <div className="sidebar-section-title">Management</div>
                <div
                    className={`nav-item ${pathname === '/domains' ? 'active' : ''}`}
                    onClick={() => navigate('/domains')}
                >
                    <span className="icon">🌐</span>
                    <span>Domains</span>
                </div>
                <div
                    className={`nav-item ${pathname === '/settings' ? 'active' : ''}`}
                    onClick={() => navigate('/settings')}
                >
                    <span className="icon">⚙️</span>
                    <span>Settings</span>
                </div>
            </nav>

            {/* User Footer */}
            <div className="sidebar-footer">
                <div className="user-info" onClick={() => navigate('/settings')}>
                    <div className="user-avatar">{initials}</div>
                    <div className="user-details">
                        <div className="name">{user?.display_name || 'User'}</div>
                        <div className="email">{user?.email || ''}</div>
                    </div>
                </div>
                <button
                    className="btn btn-ghost btn-sm"
                    style={{ width: '100%', marginTop: '4px' }}
                    onClick={handleLogout}
                >
                    🚪 Sign Out
                </button>
            </div>
        </aside>
    );
}
