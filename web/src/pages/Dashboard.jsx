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
    const [alpha, setAlpha] = useState(0); // Compass heading
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

    // Device Orientation Handler
    useEffect(() => {
        const handleOrientation = (e) => {
            let heading = 0;
            // Webkit (iOS) specific
            if (e.webkitCompassHeading) {
                heading = e.webkitCompassHeading;
            } else if (e.alpha !== null) {
                // Android/Standard: alpha is 0 at North? Not always, but close assumption for web relative
                // Or simply: alpha increases counter-clockwise.
                // Compass heading approx = 360 - alpha
                heading = 360 - e.alpha;
            }
            setAlpha(heading);
        };

        if (isActive) {
            window.addEventListener('deviceorientation', handleOrientation);
        }
        return () => {
            window.removeEventListener('deviceorientation', handleOrientation);
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
            // Request permission for iOS
            if (typeof DeviceOrientationEvent !== 'undefined' && typeof DeviceOrientationEvent.requestPermission === 'function') {
                try {
                    const permission = await DeviceOrientationEvent.requestPermission();
                    console.log('Orientation permission:', permission);
                } catch (e) {
                    console.warn(e);
                }
            }
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

                    // AR / Relative Positioning Logic
                    // 1. Calculate Bearing
                    const myLat = user?.lat;
                    const myLong = user?.long;
                    const targetLat = match.lat;
                    const targetLong = match.long;

                    let leftPos = '50%';
                    let topPos = '50%';
                    let isVisible = true;

                    if (myLat && myLong && targetLat && targetLong) {
                        const toRad = (deg) => deg * Math.PI / 180;
                        const toDeg = (rad) => rad * 180 / Math.PI;

                        const phi1 = toRad(myLat);
                        const phi2 = toRad(targetLat);
                        const dLam = toRad(targetLong - myLong);

                        const y = Math.sin(dLam) * Math.cos(phi2);
                        const x = Math.cos(phi1) * Math.sin(phi2) -
                            Math.sin(phi1) * Math.cos(phi2) * Math.cos(dLam);

                        let bearing = toDeg(Math.atan2(y, x));
                        bearing = (bearing + 360) % 360; // Normalize to 0-360

                        // 2. Compare with device azimuth (alpha)
                        // alpha is where phone is pointing (accumulated from state)
                        // If phone points 0 (N), and bearing is 90 (E), delta is +90 (Right of screen)
                        let delta = bearing - alpha;

                        // Shortest path scaling
                        if (delta < -180) delta += 360;
                        if (delta > 180) delta -= 360;

                        // 3. Map to screen
                        // Field of View (FOV) = 90 degrees?
                        // Center (0 delta) = 50% left
                        // -45 deg = 0% left
                        // +45 deg = 100% left
                        const fov = 100; // slightly wider than 90
                        const screenX = 50 + (delta / (fov / 2)) * 50;

                        leftPos = `${screenX}%`;

                        // Fade out if out of view
                        if (screenX < -10 || screenX > 110) {
                            isVisible = false;
                        }

                        // Distance affects Top (Perspective)
                        // Closer = Lower (bottom of screen), Further = Higher (closer to horizon)
                        // distance_ft mapping:
                        // 0 ft = 80% top (very close)
                        // 10000 miles... map logs?
                        // Simple linear clamp for demo:
                        // Max view distance 5000 miles?
                        // Just somewhat random Y based on index if no distance logic perfectly fits screen
                        // But let's try mapping inverse distance to Y.
                        // Actually, simplified: keep Y spread to avoid overlap, but X is strictly directional.
                        // Or use distance for scale mainly.
                        topPos = `${30 + (index * 10)}%`; // Keep Y separate to prevent overlapping text
                    } else {
                        // Fallback if no location data
                        const positions = [
                            { top: '35%', left: '15%' },
                            { top: '45%', right: '20%' },
                            { bottom: '20%', left: '25%' },
                            { bottom: '25%', right: '15%' }
                        ];
                        const pos = positions[index % positions.length];
                        if (pos.left) leftPos = pos.left;
                        if (pos.right) leftPos = `calc(100% - ${pos.right})`;
                        if (pos.top) topPos = pos.top;
                        if (pos.bottom) topPos = `calc(100% - ${pos.bottom})`;
                    }

                    const scoreNormalized = Math.min(match.matching_score / 100, 1);
                    const size = 60 + (scoreNormalized * 140);
                    const brightness = 0.4 + (scoreNormalized * 0.6);

                    if (!isVisible) return null;

                    return (
                        <div
                            key={match.user_id}
                            className="match-orb"
                            onClick={() => navigate(`/user/${match.user_id}`)}
                            style={{
                                left: leftPos,
                                top: topPos,
                                position: 'absolute',
                                transform: 'translate(-50%, -50%)', // Center on coordinate
                                width: `${size}px`,
                                height: `${size}px`,
                                opacity: brightness,
                                background: `radial-gradient(circle at 30% 30%, rgba(255,255,255,0.9) 0%, ${color} 25%, ${color}ee 60%, ${color}99 100%)`,
                                boxShadow: `
                                    0 0 20px ${color},
                                    0 0 60px ${color}aa,
                                    0 0 100px ${color}66
                                `,
                                transition: 'left 0.2s ease-out, top 0.5s ease-out', // Smooth movement
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
                            {typeof match.distance_ft === 'number' && (
                                <span className="orb-distance" style={{ fontSize: '0.8rem', marginTop: '2px', color: 'white', textShadow: '0 1px 2px rgba(0,0,0,0.5)' }}>
                                    {Math.round(match.distance_ft).toLocaleString()} ft
                                </span>
                            )}
                        </div>
                    );
                })}
            </div>

            <div className="glow-orb" style={{ top: '50%', width: '800px', height: '800px', opacity: 0.2 }}></div>
        </div>
    );
};

export default Dashboard;
