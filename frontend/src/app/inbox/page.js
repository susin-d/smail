'use client';

import { useEffect, useState, useCallback, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Sidebar from '@/components/Sidebar';
import api from '@/lib/api';

function InboxContent() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const folder = searchParams.get('folder') || 'INBOX';

    const [mails, setMails] = useState([]);
    const [total, setTotal] = useState(0);
    const [page, setPage] = useState(1);
    const [loading, setLoading] = useState(true);

    const fetchInbox = useCallback(async () => {
        setLoading(true);
        try {
            const data = await api.getInbox(folder, page, 50);
            setMails(data.mails || []);
            setTotal(data.total || 0);
        } catch {
            setMails([]);
        } finally {
            setLoading(false);
        }
    }, [folder, page]);

    useEffect(() => {
        fetchInbox();
    }, [fetchInbox]);

    const handleStarToggle = async (e, mail) => {
        e.stopPropagation();
        try {
            await api.mailAction(mail.id, mail.is_starred ? 'unstar' : 'star');
            setMails(prev =>
                prev.map(m =>
                    m.id === mail.id ? { ...m, is_starred: !m.is_starred } : m
                )
            );
        } catch (err) {
            console.error(err);
        }
    };

    const formatTime = (timestamp) => {
        const date = new Date(timestamp);
        const now = new Date();
        const diffMs = now - date;
        const diffHours = diffMs / (1000 * 60 * 60);

        if (diffHours < 24) {
            return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        }
        if (diffHours < 168) {
            return date.toLocaleDateString([], { weekday: 'short' });
        }
        return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
    };

    const folderTitle = folder === 'INBOX' ? 'Inbox' : folder;

    return (
        <>
            <div className="page-header">
                <h2>{folderTitle} <span style={{ fontSize: 14, color: 'var(--text-muted)', fontWeight: 400 }}>({total})</span></h2>
                <div className="actions">
                    <button className="btn btn-ghost btn-sm" onClick={fetchInbox}>🔄 Refresh</button>
                </div>
            </div>

            <div className="glass-card" style={{ padding: 0, overflow: 'hidden' }}>
                {loading ? (
                    <div className="loading-container">
                        <div className="spinner" />
                    </div>
                ) : mails.length === 0 ? (
                    <div className="empty-state">
                        <div className="icon">📭</div>
                        <h3>No emails</h3>
                        <p>Your {folderTitle.toLowerCase()} is empty</p>
                    </div>
                ) : (
                    <div className="mail-list">
                        {mails.map((mail) => (
                            <div
                                key={mail.id}
                                className={`mail-item ${!mail.is_read ? 'unread' : ''}`}
                                onClick={() => router.push(`/mail/view?id=${mail.id}`)}
                            >
                                <input
                                    type="checkbox"
                                    className="mail-checkbox"
                                    onClick={(e) => e.stopPropagation()}
                                />
                                <span
                                    className={`mail-star ${mail.is_starred ? 'starred' : ''}`}
                                    onClick={(e) => handleStarToggle(e, mail)}
                                >
                                    {mail.is_starred ? '★' : '☆'}
                                </span>
                                <span className="mail-sender">{mail.sender}</span>
                                <div className="mail-content">
                                    <span className="mail-subject">{mail.subject || '(no subject)'}</span>
                                </div>
                                <div className="mail-meta">
                                    {mail.has_attachments && <span className="mail-attachment">📎</span>}
                                    <span className="mail-time">{formatTime(mail.timestamp)}</span>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>

            {/* Pagination */}
            {total > 50 && (
                <div style={{ display: 'flex', justifyContent: 'center', gap: 10, marginTop: 20 }}>
                    <button
                        className="btn btn-secondary btn-sm"
                        disabled={page <= 1}
                        onClick={() => setPage(p => p - 1)}
                    >
                        ← Previous
                    </button>
                    <span style={{ padding: '6px 12px', color: 'var(--text-muted)', fontSize: 13 }}>
                        Page {page}
                    </span>
                    <button
                        className="btn btn-secondary btn-sm"
                        disabled={page * 50 >= total}
                        onClick={() => setPage(p => p + 1)}
                    >
                        Next →
                    </button>
                </div>
            )}
        </>
    );
}

export default function InboxPage() {
    return (
        <div className="app-layout">
            <Sidebar />
            <main className="main-content">
                <Suspense fallback={
                    <div className="loading-container">
                        <div className="spinner" />
                    </div>
                }>
                    <InboxContent />
                </Suspense>
            </main>
        </div>
    );
}
