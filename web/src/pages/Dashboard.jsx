import { useNavigate } from "react-router-dom";
import { useEffect, useState } from "react";

const Dashboard = () => {
    const navigate = useNavigate();
    const [loading, setLoading] = useState(true);
    const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

    useEffect(() => {
        const checkAuth = async () => {
            try {
                const response = await fetch(`${apiBase}/api/me`);
                if (!response.ok) {
                    throw new Error("Unauthorized");
                }
                setLoading(false);
            } catch (error) {
                console.error("Dashboard auth check failed:", error);
                navigate("/login");
            }
        };
        checkAuth();
    }, [navigate]);

    const handleLogout = () => {
        // Clear tokens/state if stored client side
        document.cookie = "access_token=; Max-Age=0; path=/";
        document.cookie = "refresh_token=; Max-Age=0; path=/";
        navigate("/login");
    };

    if (loading) {
        return (
            <div className="app center-content">
                <div className="glow-orb" style={{ width: '200px', height: '200px' }}></div>
                <p className="glow-text">Loading...</p>
            </div>
        );
    }

    return (
        <div className="app">
            <header className="header" style={{ position: "relative" }}>
                <div className="container header-content">
                    <div className="logo glow-text">GlowMeet</div>
                    <nav className="nav">
                        <button className="btn-ghost" onClick={handleLogout}>
                            Logout
                        </button>
                    </nav>
                </div>
            </header>

            <div className="container" style={{ padding: "40px 20px" }}>
                <h1 className="hero-title" style={{ fontSize: "2.5rem", marginBottom: "30px" }}>
                    Welcome Back <span className="glow-text">Creator</span>
                </h1>

                <div className="glass-card">
                    <p style={{ color: "var(--color-text-muted)" }}>
                        You are now logged in. This is where your personalized feed and matching will happen.
                    </p>
                </div>
            </div>
            <div className="glow-orb" style={{ top: '20%', width: '500px', height: '500px' }}></div>
        </div>
    );
};

export default Dashboard;
