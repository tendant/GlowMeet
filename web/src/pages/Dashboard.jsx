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
            intervalId = setInterval(fetchMatches, 20 * 1000); // 10 minutes
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
                    const colors = ['#FF6B6B', '#4ECDC4', '#FFD93D', '#A8E6CF'];
                    const color = colors[index % colors.length];

                    // Random positioning using deterministic hash from user_id
                    const hash = match.user_id.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
                    const leftPos = `${10 + ((hash * 17) % 80)}%`;
                    const topPos = `${15 + ((hash * 23) % 70)}%`;

                    const scoreNormalized = Math.min(match.matching_score / 100, 1);
                    const size = 80 + (scoreNormalized * 80);
                    const brightness = 0.5 + (scoreNormalized * 0.5);

                    // Random animation delay for heartbeat
                    const animDelay = (hash % 20) * 0.1;

                    return (
                        <div
                            key={match.user_id}
                            className="match-orb"
                            onClick={() => navigate(`/user/${match.user_id}`)}
                            style={{
                                left: leftPos,
                                top: topPos,
                                position: 'absolute',
                                transform: 'translate(-50%, -50%)',
                                width: `${size}px`,
                                height: `${size}px`,
                                opacity: brightness,
                                background: `radial-gradient(circle at 30% 30%, rgba(255,255,255,0.9) 0%, ${color} 25%, ${color}ee 60%, ${color}99 100%)`,
                                boxShadow: `
                                    0 0 20px ${color},
                                    0 0 60px ${color}aa,
                                    0 0 100px ${color}66
                                `,
                                animation: `orb-heartbeat 2s ease-in-out infinite`,
                                animationDelay: `${animDelay}s`,
                                zIndex: Math.round(scoreNormalized * 10)
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
