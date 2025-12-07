import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import Header from '../components/Header';
import './UserDetails.css';

const UserDetails = () => {
    const { userId } = useParams();
    const navigate = useNavigate();
    const [user, setUser] = useState(null);
    const [loading, setLoading] = useState(true);

    // Mock data based on userId for now, since we only have list endpoint not individual yet or simplified
    useEffect(() => {
        // Simulating fetch
        const fetchUser = async () => {
            // In a real app, fetch `${apiBase}/api/users/${userId}`
            // For now, we'll just mock it based on the ID to show the UI
            setTimeout(() => {
                setUser({
                    id: userId,
                    name: `User ${userId.substring(0, 4)}...`, // Placeholder
                    username: `@user${userId.substring(0, 4)}`,
                    summary: "Loves photography, hiking, and exploring new coffee shops. Always looking for the next great adventure and someone to share it with.",
                    bgImage: `https://picsum.photos/seed/${userId}/800/400`,
                    tweets: [
                        "Just saw the most amazing sunset! 🌅 #nature #vibes",
                        "Coffee is life. Anyone know a good spot in downtown?",
                        "Thinking about my next trip. Japan or Iceland? ✈️",
                        "Coding late into the night. The flow is real."
                    ]
                });
                setLoading(false);
            }, 500);
        };
        fetchUser();
    }, [userId]);

    const goBack = () => navigate(-1);

    if (loading) {
        return (
            <div className="user-details-container center-content">
                <div className="glow-orb" style={{ width: '100px', height: '100px' }}></div>
            </div>
        );
    }

    return (
        <div className="user-details-container">
            <header className="details-header">
                <button onClick={goBack} className="back-btn">
                    ← Back
                </button>
            </header>

            <main className="details-content">
                <section className="details-hero">
                    <div className="ai-bg-card">
                        <span className="ai-badge">AI Insight</span>
                        <img src={user?.bgImage} alt="AI Generated Background" className="bg-image" />
                        <h1 className="details-username-hero">{user?.username}</h1>
                        <p className="ai-summary">
                            {user?.summary}
                        </p>
                    </div>
                </section>

                <section className="twitter-feed">
                    {user?.tweets.map((tweet, index) => (
                        <div key={index} className="feed-item">
                            <p>{tweet}</p>
                            <div className="feed-meta">
                                <span>Twitter Feed {index + 1}</span>
                            </div>
                        </div>
                    ))}
                </section>
            </main>
        </div>
    );
};

export default UserDetails;
