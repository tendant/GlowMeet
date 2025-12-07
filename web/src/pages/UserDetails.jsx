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
        const fetchUser = async () => {
            try {
                const response = await fetch(`${apiBase}/api/users/${userId}`);
                if (response.ok) {
                    const data = await response.json();
                    setUser({
                        ...data,
                        bgImage: data.bg_image, // Map snake_case to camelCase
                        profileImage: data.profile_image_url
                    });
                } else {
                    console.error("Failed to fetch user details");
                }
            } catch (error) {
                console.error("Error fetching user details:", error);
            } finally {
                setLoading(false);
            }
        };

        if (userId) {
            fetchUser();
        }
    }, [userId, apiBase]);

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
