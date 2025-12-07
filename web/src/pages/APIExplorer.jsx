import { useState } from 'react';
import './APIExplorer.css'; // We'll assume some basic styles or reuse existing ones

const APIExplorer = () => {
    const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');
    const [method, setMethod] = useState('GET');
    const [endpoint, setEndpoint] = useState('/api/me');
    const [body, setBody] = useState('');
    const [response, setResponse] = useState(null);
    const [loading, setLoading] = useState(false);

    const handleSend = async () => {
        setLoading(true);
        setResponse(null);
        try {
            const options = {
                method,
                headers: {}
            };

            if (method !== 'GET' && method !== 'HEAD' && body) {
                options.headers['Content-Type'] = 'application/json';
                options.body = body;
            }

            const res = await fetch(`${apiBase}${endpoint}`, options);
            const status = res.status;
            let data;
            const contentType = res.headers.get("content-type");
            if (contentType && contentType.indexOf("application/json") !== -1) {
                data = await res.json();
            } else {
                data = await res.text();
            }

            setResponse({ status, data });
        } catch (error) {
            setResponse({ error: error.message });
        } finally {
            setLoading(false);
        }
    };

    const loadPreset = (presetMethod, presetEndpoint, presetBody) => {
        setMethod(presetMethod);
        setEndpoint(presetEndpoint);
        setBody(presetBody ? JSON.stringify(presetBody, null, 2) : '');
        setResponse(null);
    };

    return (
        <div className="api-explorer-container" style={{ padding: '20px', color: 'white', maxWidth: '800px', margin: '0 auto' }}>
            <h1>API Explorer</h1>

            <div className="presets" style={{ marginBottom: '20px' }}>
                <h3>Presets:</h3>
                <button onClick={() => loadPreset('GET', '/api/me')}>GET /api/me</button>
                <button onClick={() => loadPreset('POST', '/api/me', { lat: 37.7749, long: -122.4194 })}>POST /api/me</button>
                <button onClick={() => loadPreset('GET', '/api/users')}>GET /api/users</button>
            </div>

            <div className="request-builder" style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                <div style={{ display: 'flex', gap: '10px' }}>
                    <select value={method} onChange={(e) => setMethod(e.target.value)} style={{ padding: '5px' }}>
                        <option value="GET">GET</option>
                        <option value="POST">POST</option>
                        <option value="PUT">PUT</option>
                        <option value="DELETE">DELETE</option>
                    </select>
                    <input
                        type="text"
                        value={endpoint}
                        onChange={(e) => setEndpoint(e.target.value)}
                        style={{ flex: 1, padding: '5px' }}
                        placeholder="/api/..."
                    />
                    <button onClick={handleSend} disabled={loading} style={{ padding: '5px 15px' }}>
                        {loading ? 'Sending...' : 'Send'}
                    </button>
                </div>

                {method !== 'GET' && (
                    <textarea
                        value={body}
                        onChange={(e) => setBody(e.target.value)}
                        rows={5}
                        placeholder='{"key": "value"}'
                        style={{ width: '100%', fontFamily: 'monospace', padding: '5px' }}
                    />
                )}
            </div>

            <div className="response-viewer" style={{ marginTop: '20px', borderTop: '1px solid #444', paddingTop: '20px' }}>
                <h3>Response</h3>
                {response ? (
                    <div style={{ background: '#1e1e1e', padding: '10px', borderRadius: '5px' }}>
                        <div style={{ marginBottom: '10px', fontWeight: 'bold', color: response.status >= 400 ? '#ff5555' : '#55ff55' }}>
                            Status: {response.status}
                        </div>
                        <pre style={{ margin: 0, overflowX: 'auto', color: '#d4d4d4' }}>
                            {typeof response.data === 'object' ? JSON.stringify(response.data, null, 2) : response.data}
                            {response.error && `Error: ${response.error}`}
                        </pre>
                    </div>
                ) : (
                    <p style={{ color: '#888' }}>No response yet</p>
                )}
            </div>
        </div>
    );
};

export default APIExplorer;
