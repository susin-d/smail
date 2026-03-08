'use client';

import { useEffect, useState } from 'react';
import Sidebar from '@/components/Sidebar';
import api from '@/lib/api';

export default function DomainsPage() {
    const [domains, setDomains] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showAdd, setShowAdd] = useState(false);
    const [newDomain, setNewDomain] = useState('');
    const [adding, setAdding] = useState(false);
    const [error, setError] = useState('');
    const [dnsRecords, setDnsRecords] = useState(null);

    useEffect(() => {
        fetchDomains();
    }, []);

    const fetchDomains = async () => {
        setLoading(true);
        try {
            const data = await api.getDomains();
            setDomains(data || []);
        } catch {
            setDomains([]);
        } finally {
            setLoading(false);
        }
    };

    const handleAddDomain = async (e) => {
        e.preventDefault();
        setError('');
        setAdding(true);
        try {
            const result = await api.createDomain(newDomain);
            setDnsRecords(result);
            setNewDomain('');
            fetchDomains();
        } catch (err) {
            setError(err.message);
        } finally {
            setAdding(false);
        }
    };

    const handleViewDns = async (domainId) => {
        try {
            const data = await api.getDomainDns(domainId);
            setDnsRecords(data);
        } catch (err) {
            console.error(err);
        }
    };

    const handleDelete = async (domainId) => {
        if (!confirm('Delete this domain and all its users?')) return;
        try {
            await api.deleteDomain(domainId);
            fetchDomains();
        } catch (err) {
            console.error(err);
        }
    };

    return (
        <div className="app-layout">
            <Sidebar />
            <main className="main-content">
                <div className="page-header">
                    <h2>Domains</h2>
                    <button className="btn btn-primary btn-sm" onClick={() => setShowAdd(!showAdd)}>
                        {showAdd ? '✕ Cancel' : '+ Add Domain'}
                    </button>
                </div>

                {/* Add Domain Form */}
                {showAdd && (
                    <div className="glass-card" style={{ marginBottom: 20 }}>
                        <h3 style={{ fontSize: 16, marginBottom: 12 }}>Add New Domain</h3>
                        {error && <div className="auth-error">{error}</div>}
                        <form onSubmit={handleAddDomain} style={{ display: 'flex', gap: 10, alignItems: 'end' }}>
                            <div className="form-group" style={{ flex: 1, marginBottom: 0 }}>
                                <label className="form-label">Domain Name</label>
                                <input
                                    className="form-input"
                                    type="text"
                                    placeholder="example.com"
                                    value={newDomain}
                                    onChange={(e) => setNewDomain(e.target.value)}
                                    required
                                />
                            </div>
                            <button className="btn btn-primary" type="submit" disabled={adding}>
                                {adding ? 'Adding...' : 'Add'}
                            </button>
                        </form>
                    </div>
                )}

                {/* DNS Records Modal */}
                {dnsRecords && (
                    <div className="glass-card" style={{ marginBottom: 20, borderColor: 'var(--accent)', borderWidth: 1 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                            <h3 style={{ fontSize: 16 }}>DNS Records for {dnsRecords.domain}</h3>
                            <button className="btn btn-ghost btn-sm" onClick={() => setDnsRecords(null)}>✕</button>
                        </div>
                        <p style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 12 }}>
                            Configure the following DNS records with your domain registrar:
                        </p>
                        {dnsRecords.records.map((rec, i) => (
                            <div key={i} className="dns-record">
                                <span className="type">{rec.record_type}</span>
                                <span style={{ color: 'var(--text-secondary)', flex: 1 }}>{rec.name}</span>
                                <span style={{ color: 'var(--text-muted)', maxWidth: 400, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                                    {rec.priority ? `${rec.priority} ` : ''}{rec.value}
                                </span>
                            </div>
                        ))}
                    </div>
                )}

                {/* Domain List */}
                {loading ? (
                    <div className="loading-container"><div className="spinner" /></div>
                ) : domains.length === 0 ? (
                    <div className="glass-card">
                        <div className="empty-state">
                            <div className="icon">🌐</div>
                            <h3>No domains</h3>
                            <p>Add your first domain to start receiving emails</p>
                        </div>
                    </div>
                ) : (
                    <div className="domain-grid">
                        {domains.map((domain) => (
                            <div key={domain.id} className="domain-card">
                                <div className="domain-card-header">
                                    <span className="domain-name">{domain.domain}</span>
                                    <span className={`domain-status ${domain.is_verified ? 'verified' : 'pending'}`}>
                                        {domain.is_verified ? 'Verified' : 'Pending'}
                                    </span>
                                </div>
                                <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 12 }}>
                                    Added {new Date(domain.created_at).toLocaleDateString()}
                                </div>
                                <div style={{ display: 'flex', gap: 8 }}>
                                    <button className="btn btn-secondary btn-sm" onClick={() => handleViewDns(domain.id)}>
                                        🔧 DNS Records
                                    </button>
                                    <button className="btn btn-danger btn-sm" onClick={() => handleDelete(domain.id)}>
                                        🗑️ Delete
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </main>
        </div>
    );
}
