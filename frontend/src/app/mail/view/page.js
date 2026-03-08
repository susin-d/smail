'use client';

import { useEffect, useState, useMemo, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import DOMPurify from 'dompurify';
import Sidebar from '@/components/Sidebar';
import api from '@/lib/api';

function MailViewContent() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const mailId = searchParams.get('id');

    const [mail, setMail] = useState(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        if (!mailId) return;
        setLoading(true);
        api.getMail(mailId)
            .then(setMail)
            .catch(() => router.push('/inbox'))
            .finally(() => setLoading(false));
    }, [mailId, router]);

    const handleAction = async (action) => {
        try {
            await api.mailAction(mailId, action);
            if (action === 'delete') {
                router.push('/inbox');
            } else {
                const updated = await api.getMail(mailId);
                setMail(updated);
            }
        } catch (err) {
            console.error(err);
        }
    };

    const formatDate = (ts) => {
        return new Date(ts).toLocaleString([], {
            weekday: 'long',
            year: 'numeric',
            month: 'long',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
        });
    };

    const senderInitials = mail?.sender
        ? mail.sender.split('@')[0].slice(0, 2).toUpperCase()
        : '??';

    return (
        <div className="app-layout">
            <Sidebar />
            <main className="main-content">
                {loading ? (
                    <div className="loading-container">
                        <div className="spinner" />
                    </div>
                ) : !mail ? (
                    <div className="empty-state">
                        <div className="icon">📭</div>
                        <h3>Email not found</h3>
                    </div>
                ) : (
                    <div className="mail-view">
                        {/* Toolbar */}
                        <div style={{ display: 'flex', gap: 8, marginBottom: 20 }}>
                            <button className="btn btn-ghost btn-sm" onClick={() => router.push('/inbox')}>
                                ← Back
                            </button>
                            <button className="btn btn-ghost btn-sm" onClick={() => handleAction(mail.is_read ? 'unread' : 'read')}>
                                {mail.is_read ? '📩 Mark Unread' : '📨 Mark Read'}
                            </button>
                            <button className="btn btn-ghost btn-sm" onClick={() => handleAction(mail.is_starred ? 'unstar' : 'star')}>
                                {mail.is_starred ? '★ Unstar' : '☆ Star'}
                            </button>
                            <button className="btn btn-ghost btn-sm" onClick={() => handleAction('delete')}>
                                🗑️ Delete
                            </button>
                        </div>

                        {/* Header */}
                        <div className="glass-card">
                            <h1 className="mail-view-subject">{mail.subject || '(no subject)'}</h1>

                            <div className="mail-view-header">
                                <div className="mail-view-participants">
                                    <div className="participant-avatar">{senderInitials}</div>
                                    <div className="participant-info">
                                        <div className="sender-name">{mail.sender}</div>
                                        <div className="sender-email">
                                            To: {mail.recipient} · {formatDate(mail.timestamp)}
                                        </div>
                                    </div>
                                </div>
                            </div>

                            {/* Body */}
                            <div className="mail-view-body">
                                {mail.html_body ? (
                                    <div dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(mail.html_body, { FORBID_TAGS: ['style', 'form'], FORBID_ATTR: ['onerror', 'onload', 'onclick'] }) }} />
                                ) : (
                                    mail.body.split('\n').map((line, i) => (
                                        <p key={i}>{line || '\u00A0'}</p>
                                    ))
                                )}
                            </div>

                            {/* Attachments */}
                            {mail.attachments && mail.attachments.length > 0 && (
                                <div style={{ marginTop: 24, paddingTop: 16, borderTop: '1px solid var(--border)' }}>
                                    <h4 style={{ fontSize: 14, marginBottom: 10 }}>
                                        📎 Attachments ({mail.attachments.length})
                                    </h4>
                                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                                        {mail.attachments.map((att, i) => (
                                            <div
                                                key={i}
                                                style={{
                                                    padding: '8px 14px',
                                                    background: 'var(--bg-primary)',
                                                    borderRadius: 'var(--radius-sm)',
                                                    fontSize: 13,
                                                    display: 'flex',
                                                    alignItems: 'center',
                                                    gap: 6,
                                                }}
                                            >
                                                📄 {att.filename}
                                                <span style={{ color: 'var(--text-muted)', fontSize: 11 }}>
                                                    ({(att.size / 1024).toFixed(1)} KB)
                                                </span>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}

                            {/* Reply Actions */}
                            <div style={{ marginTop: 24, display: 'flex', gap: 8 }}>
                                <button
                                    className="btn btn-secondary btn-sm"
                                    onClick={() => router.push(`/compose?replyTo=${encodeURIComponent(mail.sender)}&subject=${encodeURIComponent('Re: ' + (mail.subject || ''))}`)}
                                >
                                    ↩️ Reply
                                </button>
                                <button
                                    className="btn btn-secondary btn-sm"
                                    onClick={() => router.push(`/compose?subject=${encodeURIComponent('Fwd: ' + (mail.subject || ''))}`)}
                                >
                                    ↪️ Forward
                                </button>
                            </div>
                        </div>
                    </div>
                )}
            </main>
        </div>
    );
}

export default function MailViewPage() {
    return (
        <Suspense fallback={<div className="loading-container"><div className="spinner" /></div>}>
            <MailViewContent />
        </Suspense>
    );
}
