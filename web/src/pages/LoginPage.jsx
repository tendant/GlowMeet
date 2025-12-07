import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

import './LoginPage.css';

const XIcon = () => (
    <svg viewBox="0 0 24 24" aria-hidden="true" width="20" height="20" fill="currentColor" style={{ marginRight: '10px' }}>
        <g>
            <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"></path>
        </g>
    </svg>
);

const LoginPage = ({ mode = 'login' }) => {
    const navigate = useNavigate();
    const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

    const handleXAuth = async () => {
        try {
            const response = await fetch(`${apiBase}/auth/x/login`);
            const data = await response.json();

            if (data.authorization_url) {
                // Redirect the user to X.com
                window.location.href = data.authorization_url;
            } else {
                console.error("No authorization URL received:", data);
                alert("Failed to initiate X login.");
            }
        } catch (error) {
            console.error("Error fetching auth URL:", error);
            alert("Error connecting to backend server.");
        }
    };

    return (
        <div className="app login-container">
            <header className="header login-header">
                <div className="container header-content">
                    <div className="logo glow-text" onClick={() => navigate('/')} style={{ cursor: 'pointer' }}>GlowMeet</div>
                </div>
            </header>

            <div className="login-content">
                <div className="glass-card login-card">
                    <h2 className="login-title">
                        {mode === 'signup' ? 'Join GlowMeet' : 'Welcome Back'}
                    </h2>
                    <p className="login-description">
                        {mode === 'signup'
                            ? 'Connect with people nearby who share your vibe.'
                            : 'Sign in to continue your journey.'}
                    </p>

                    <button
                        className="x-auth-button"
                        onClick={handleXAuth}
                    >
                        <XIcon />
                        Sign {mode === 'signup' ? 'up' : 'in'} with X
                    </button>

                    <div className="login-footer">
                        {mode === 'signup' ? (
                            <>Already have an account? <span onClick={() => navigate('/login')} className="link-text">Log in</span></>
                        ) : (
                            <>Don't have an account? <span onClick={() => navigate('/signup')} className="link-text">Sign up</span></>
                        )}
                    </div>
                </div>
            </div>
            <div className="glow-orb login-orb"></div>
        </div>
    );
};

export default LoginPage;
