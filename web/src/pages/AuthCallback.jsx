import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

const AuthCallback = () => {
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();

    useEffect(() => {
        const code = searchParams.get('code');
        const state = searchParams.get('state');

        if (code) {
            // Here you would typically send the code to your backend
            // await fetch(`${import.meta.env.VITE_API_BASE_URL}/auth/callback`, { method: 'POST', body: JSON.stringify({ code }) })
            console.log("Received Auth Code:", code);

            // Simulate successful login
            setTimeout(() => {
                navigate('/dashboard'); // Navigate to main app area
            }, 1500);
        } else {
            // Handle error or cancellation
            navigate('/login');
        }
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
