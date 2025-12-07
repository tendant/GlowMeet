import { useState, useContext, useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import UserContext from "../context/UserContext";
import Header from "../components/Header";
import "./Dashboard.css";

const Dashboard = () => {
    const navigate = useNavigate();
    const location = useLocation();
    const user = useContext(UserContext);

    // Initialize state from navigation state if available (e.g. coming back from UserDetails)
    const [isActive, setIsActive] = useState(location.state?.active || true);
    const [matches, setMatches] = useState([]);
    const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

    // Fetch matches immediately and then every 10 minutes if we're starting in active state
    useEffect(() => {
        let intervalId;
        if (isActive) {
            fetchMatches();
            intervalId = setInterval(fetchMatches, 10 * 1000); // 10 minutes
        }
        return () => {
            if (intervalId) clearInterval(intervalId);
        };
    }, [isActive]);

    const fetchMatches = async () => {
        let sorted = [];
        try {
            const response = await fetch(`${apiBase}/api/users`);
            if (response.ok) {
                const users = await response.json();
                // Get top 4 by matching_score
                sorted = users
                    .filter(u => u.user_id !== user?.id) // Exclude self
                    .sort((a, b) => b.matching_score - a.matching_score);
            }
        } catch (error) {
            console.error("[Matches] Failed to fetch:", error);
        }

        // Fallback Mock Data for Testing if no real matches found
        if (sorted.length === 0) {
            sorted = [
                { user_id: 'mock1', name: 'Alice', matching_score: 95 },
                { user_id: 'mock2', name: 'Bob', matching_score: 88 },
                { user_id: 'mock3', name: 'Charlie', matching_score: 75 },
                { user_id: 'mock4', name: 'Diana', matching_score: 60 }
            ];
        }

        setMatches(sorted.slice(0, 4));
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



                    return (
                        <div
                            key={match.user_id}
                            className="match-orb"
                            onClick={() => navigate(`/user/${match.user_id}`)}
                            style={{
                                ...position,
                                width: `${size}px`,
                                height: `${size}px`,
                                opacity: brightness,
                                // Ethereal Memory Orb Style
                                background: `radial-gradient(circle at 30% 30%, rgba(255,255,255,0.9) 0%, ${color} 25%, ${color}ee 60%, ${color}99 100%)`,
                                boxShadow: `
                                    0 0 20px ${color},
                                    0 0 60px ${color}aa,
                                    0 0 100px ${color}66,
                                    inset 10px 10px 20px rgba(255,255,255,0.5),
                                    inset -10px -10px 20px rgba(0,0,0,0.1)
                                `,
                                animationDelay,
                                animationDuration
                            }}
                            title={match.name || match.username}
                        >
                            <span className="orb-username">
                                {match.name || match.username}
                            </span>
                            <span className="orb-score">
                                {Math.round(match.matching_score)}%
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
