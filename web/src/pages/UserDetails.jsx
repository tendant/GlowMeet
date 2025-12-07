import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import Header from '../components/Header';
import './UserDetails.css';

const UserDetails = () => {
    const { userId } = useParams();
    const navigate = useNavigate();
    const [user, setUser] = useState(null);
    const [loading, setLoading] = useState(true);

    const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

    useEffect(() => {
        if (!userId) return;

        let isMounted = true;
        setLoading(true);
        setUser(null); // Clear previous user state

        const fetchUser = async () => {
            try {
                const response = await fetch(`${apiBase}/api/users/${userId}`);
                if (response.ok) {
                    const data = await response.json();
                    if (isMounted) {
                        setUser({
                            ...data,
                            bgImage: data.bg_image, // Map snake_case to camelCase
                            profileImage: data.profile_image_url,
                            tweets: data.tweets || []
                        });
                    }
                } else {
                    console.error("Failed to fetch user details");
                    // User remains null, will trigger "User not found" view
                }
            } catch (error) {
                console.error("Error fetching user details:", error);
            } finally {
                if (isMounted) {
                    setLoading(false);
                }
            }
        };

        fetchUser();

        return () => {
            isMounted = false;
        };
    }, [userId, apiBase]);

    const goBack = () => navigate('/', { state: { active: true } });

    if (loading) {
        return (
            <div className="user-details-container center-content">
                <div className="glow-orb" style={{ width: '100px', height: '100px' }}></div>
            </div>
        );
    }

    if (!user) {
        return (
            <div className="user-details-container center-content">
                <div className="error-message">User not found</div>
                <button onClick={goBack} className="back-btn-error">← Back</button>
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
                        <img src={user.bgImage} alt="AI Generated Background" className="bg-image" />
                        <h1 className="details-username-hero">{user.username}</h1>
                        <p className="ai-summary">
                            {user.match_info?.shared_summary || user.summary}
                        </p>
                    </div>
                </section>

                <section className="twitter-feed">
                    {user.tweets && user.tweets.length > 0 ? (
                        user.tweets.map((tweet, index) => (
                            <div key={index} className="feed-item">
                                <p>{tweet}</p>
                            </div>
                        ))
                    ) : (
                        <p className="no-tweets">No tweets available to analyze.</p>
                    )}
                </section>
            </main>
        </div>
    );
};

export default UserDetails;
