import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { isAuthenticated } from '../utils/auth';

const LandingPage = () => {
    const navigate = useNavigate();



    return (
        <>
            <header className="header">
                <div className="container header-content">
                    <div className="logo glow-text">GlowMeet</div>
                    <nav className="nav">
                        <button className="btn-primary" onClick={() => navigate('/login')}>Login</button>
                    </nav>
                </div>
            </header>

            <main>
                <section className="hero">
                    <div className="glow-orb"></div>
                    <div className="container hero-content">
                        <h1 className="hero-title">
                            Connect with <span className="glow-text">Soulmates</span> <br />
                            Nearby via AI
                        </h1>
                        <p className="hero-description">
                            GlowMeet uses advanced AI and realtime geolocation to match you with people who share your passions, right where you are.
                        </p>
                        <div className="hero-actions">
                            <button className="btn-primary btn-large" onClick={() => navigate('/login')}>Login</button>
                        </div>
                    </div>
                </section>

                <section className="features container">
                    <div className="glass-card feature-card">
                        <h3>📍 Real-time Geolocation</h3>
                        <p>Find matches within 100 meters of your current location instantly. Connect in the real world.</p>
                    </div>
                    <div className="glass-card feature-card">
                        <h3>🧠 AI Powered Matching</h3>
                        <p>Our smart algorithms analyze specific interests to ensure meaningful connections beyond surface level.</p>
                    </div>
                    <div className="glass-card feature-card">
                        <h3>🔒 Safe & Private</h3>
                        <p>Your location is blurred and only shared when you explicitly choose to connect.</p>
                    </div>
                </section>
            </main>
        </>
    )
}

export default LandingPage
