import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

const AuthCallback = () => {
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

    useEffect(() => {
        const exchangeToken = async () => {
            const code = searchParams.get('code');
            const state = searchParams.get('state');

            if (code && state) {
                try {
                    const response = await fetch(`${apiBase}/auth/x/callback?code=${code}&state=${state}`);
                    const data = await response.json();

                    if (response.ok) {
                        console.log("Login successful:", data);
                        // Store tokens (in-memory or secure storage depending on requirements)
                        // For now, we just redirect
                        navigate('/dashboard');
                    } else {
                        console.error("Login failed:", data);
                        navigate('/login');
                    }
                } catch (error) {
                    console.error("Login error:", error);
                    navigate('/login');
                }
            } else {
                navigate('/login');
            }
        };
        exchangeToken();
    }, [searchParams, navigate]);

    return (
        <div className="app" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100vh' }}>
            <div className="glass-card" style={{ textAlign: 'center' }}>
                <h2 className="glow-text">Authenticating...</h2>
                <p style={{ color: 'var(--color-text-muted)', marginTop: '1rem' }}>Please wait while we connect to X.</p>
            </div>
            <div className="glow-orb"></div>
        </div>
    );
};

export default AuthCallback;
