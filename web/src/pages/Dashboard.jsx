import { useState, useContext, useEffect } from "react";
import UserContext from "../context/UserContext";
import Header from "../components/Header";
import "./Dashboard.css";

const Dashboard = () => {
    const user = useContext(UserContext);
    const [isActive, setIsActive] = useState(false);
    const [matches, setMatches] = useState([]);
    const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

    const fetchMatches = async () => {
        try {
            const response = await fetch(`${apiBase}/api/users`);
            if (response.ok) {
                const users = await response.json();
                // Get top 4 by matching_score
                const sorted = users
                    .filter(u => u.user_id !== user?.id) // Exclude self
                    .sort((a, b) => b.matching_score - a.matching_score)
                    .slice(0, 4);
                setMatches(sorted);
            }
        } catch (error) {
            console.error("[Matches] Failed to fetch:", error);
        }
    };

    const toggleConnection = async () => {
        if (!isActive) {
            // Connecting
            setIsActive(true);
            await fetchMatches();
        } else {
            // Disconnecting
            setIsActive(false);
            setMatches([]);
        }
    };

    return (
        <div className="app dashboard-container">
            <Header user={user} showLogout={true} />

            <div className="dashboard-content">
                <div className="user-display">
                    <h1 className="large-username">{user?.name || user?.username || "Creator"}</h1>
                    <p className="connection-status" style={{ color: isActive ? '#FF8C00' : '#6B7280' }}>
                        {isActive ? "connected!" : "currently unconnected"}
                    </p>
                </div>

                <div
                    className={`connection-orb ${isActive ? 'active' : ''}`}
                    onClick={toggleConnection}
                >
                    <span className="orb-cta">
                        {isActive ? "Searching..." : "Connect Now"}
                    </span>
                </div>

                {isActive && matches.map((match, index) => {
                    const colors = ['#FF6B6B', '#4ECDC4', '#FFD93D', '#A8E6CF']; // Red, Teal, Yellow, Green
                    const positions = [
                        { top: '20%', left: '15%' },
                        { top: '30%', right: '20%' },
                        { bottom: '25%', left: '25%' },
                        { bottom: '30%', right: '15%' }
                    ];
                    const color = colors[index];
                    const position = positions[index];
                    const scoreNormalized = Math.min(match.matching_score / 100, 1); // 0-1
                    const size = 80 + (scoreNormalized * 80); // 80px - 160px
                    const brightness = 0.5 + (scoreNormalized * 0.5); // 0.5 - 1.0

                    return (
                        <div
                            key={match.user_id}
                            className="match-orb"
                            style={{
                                ...position,
                                width: `${size}px`,
                                height: `${size}px`,
                                opacity: brightness,
                                background: `radial-gradient(circle at 50% 50%, ${color}, ${color}dd)`,
                                boxShadow: `
                                    0 0 ${30 * brightness}px ${color}aa,
                                    0 0 ${60 * brightness}px ${color}77,
                                    0 0 ${90 * brightness}px ${color}44
                                `
                            }}
                            title={match.name || match.username}
                        />
                    );
                })}
            </div>

            <div className="glow-orb" style={{ top: '50%', width: '800px', height: '800px', opacity: 0.2 }}></div>
        </div>
    );
};

export default Dashboard;
