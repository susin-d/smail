'use client';

import { useState, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Sidebar from '@/components/Sidebar';
import api from '@/lib/api';

function ComposeContent() {
    const router = useRouter();
    const searchParams = useSearchParams();

    const [to, setTo] = useState(searchParams.get('replyTo') || '');
    const [subject, setSubject] = useState(searchParams.get('subject') || '');
    const [body, setBody] = useState('');
    const [sending, setSending] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState(false);

    const handleSend = async (e) => {
        e.preventDefault();
        setError('');
        setSending(true);

        try {
            await api.sendMail(to, subject, body);
            setSuccess(true);
            setTimeout(() => router.push('/inbox'), 1500);
        } catch (err) {
            setError(err.message || 'Failed to send email');
        } finally {
            setSending(false);
        }
    };

    return (
        <div className="compose-form">
            <div className="page-header">
                <h2>Compose</h2>
                <div className="actions">
                    <button className="btn btn-ghost btn-sm" onClick={() => router.back()}>
                        ✕ Discard
                    </button>
                </div>
            </div>

            {error && <div className="auth-error">{error}</div>}
            {success && (
                <div className="toast success">✓ Email queued for delivery!</div>
            )}

            <div className="glass-card">
                <form onSubmit={handleSend}>
                    <div className="compose-field">
                        <label>To</label>
                        <input
                            type="email"
                            placeholder="recipient@example.com"
                            value={to}
                            onChange={(e) => setTo(e.target.value)}
                            required
                            autoFocus={!to}
                        />
                    </div>

                    <div className="compose-field">
                        <label>Subject</label>
                        <input
                            type="text"
                            placeholder="Email subject"
                            value={subject}
                            onChange={(e) => setSubject(e.target.value)}
                            required
                            autoFocus={!!to}
                        />
                    </div>

                    <div className="compose-body">
                        <textarea
                            placeholder="Write your message..."
                            value={body}
                            onChange={(e) => setBody(e.target.value)}
                            required
                        />
                    </div>

                    <div className="compose-actions">
                        <button
                            className="btn btn-primary"
                            type="submit"
                            disabled={sending}
                        >
                            {sending ? (
                                <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                    <span className="spinner" style={{ width: 18, height: 18, borderWidth: 2 }} />
                                    Sending...
                                </span>
                            ) : '📨 Send'}
                        </button>

                        <div style={{ display: 'flex', gap: 8 }}>
                            <button type="button" className="btn btn-ghost btn-sm" title="Attach file">
                                📎 Attach
                            </button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    );
}

export default function ComposePage() {
    return (
        <div className="app-layout">
            <Sidebar />
            <main className="main-content">
                <Suspense fallback={
                    <div className="loading-container">
                        <div className="spinner" />
                    </div>
                }>
                    <ComposeContent />
                </Suspense>
            </main>
        </div>
    );
}
