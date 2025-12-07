import { useEffect, useRef } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

const AuthCallback = () => {
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const effectRan = useRef(false);



    useEffect(() => {



        if (effectRan.current) return;

        const exchangeToken = async () => {
            const code = searchParams.get('code');
            const state = searchParams.get('state');

            if (code && state) {
                effectRan.current = true;
                console.log("[AuthCallback] Exchanging code for token...");
                try {
                    const response = await fetch(`/auth/x/callback?code=${code}&state=${state}`);
                    const data = await response.json();

                    console.log("[AuthCallback] Backend response:", data);

                    if (response.ok) {
                        // Store tokens in cookies
                        if (data.access_token) {
                            const cookieString = `access_token=${data.access_token}; path=/; max-age=${data.expires_in || 3600}; SameSite=Lax`;
                            document.cookie = cookieString;
                            console.log("[AuthCallback] Set access_token cookie:", cookieString);
                        }
                        if (data.refresh_token) {
                            const cookieString = `refresh_token=${data.refresh_token}; path=/; max-age=86400; SameSite=Lax`;
                            document.cookie = cookieString;
                            console.log("[AuthCallback] Set refresh_token cookie:", cookieString);
                        }

                        console.log("[AuthCallback] Current document.cookie:", document.cookie);

                        // Small delay to verify cookies are set before navigating
                        setTimeout(() => {
                            console.log("[AuthCallback] Navigating to /dashboard");
                            navigate('/dashboard');
                        }, 500);
                    } else {
                        console.error("[AuthCallback] Login failed:", data);
                        navigate('/login');
                    }
                } catch (error) {
                    console.error("[AuthCallback] Login error:", error);
                    navigate('/login');
                }
            } else {
                navigate('/login');
            }
        };
        exchangeToken();
    }, [searchParams, navigate, apiBase]);

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
