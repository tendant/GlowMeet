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
                {!isActive && (
                    <>
                        <div className="user-display">
                            <h1 className="large-username">{user?.name || user?.username || "Creator"}</h1>
                            <p className="connection-status">
                                currently unconnected
                            </p>
                        </div>

                        <div
                            className="connection-orb"
                            onClick={toggleConnection}
                        >
                            <span className="orb-cta">
                                Connect Now
                            </span>
                        </div>
                    </>
                )}

                {isActive && (
                    <div className="connected-status">
                        <h1 className="large-username">{user?.name || user?.username || "Creator"}</h1>
                        <p className="connection-status" style={{ color: '#FF8C00', fontSize: '1.2rem', marginTop: '10px' }}>
                            connected!
                        </p>
                    </div>
                )}

                {isActive && matches.map((match, index) => {
                    const colors = ['#FF6B6B', '#4ECDC4', '#FFD93D', '#A8E6CF']; // Red, Teal, Yellow, Green
                    const positions = [
                        { top: '35%', left: '15%' },
                        { top: '45%', right: '20%' },
                        { bottom: '20%', left: '25%' },
                        { bottom: '25%', right: '15%' }
                    ];
                    const color = colors[index];
                    const position = positions[index];
                    const scoreNormalized = Math.min(match.matching_score / 100, 1); // 0-1
                    // Much bigger size range: 60px to 200px
                    const size = 60 + (scoreNormalized * 140);
                    const brightness = 0.4 + (scoreNormalized * 0.6); // 0.4 - 1.0
                    // Individual animation delay so they move independently
                    const animationDelay = `${index * 0.7}s`;
                    const animationDuration = `${2.5 + index * 0.5}s`;

                    // Darker version of color for text
                    const darkerColor = color.replace('#', '');
                    const r = parseInt(darkerColor.substr(0, 2), 16);
                    const g = parseInt(darkerColor.substr(2, 2), 16);
                    const b = parseInt(darkerColor.substr(4, 2), 16);
                    const textColor = `rgb(${Math.floor(r * 0.5)}, ${Math.floor(g * 0.5)}, ${Math.floor(b * 0.5)})`;

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
                                `,
                                animationDelay,
                                animationDuration
                            }}
                            title={match.name || match.username}
                        >
                            <span className="orb-username" style={{ color: textColor }}>
                                {match.name || match.username}
                            </span>
                        </div>
                    );
                })}
            </div>

            <div className="glow-orb" style={{ top: '50%', width: '800px', height: '800px', opacity: 0.2 }}></div>
        </div>
    );
};

export default Dashboard;
