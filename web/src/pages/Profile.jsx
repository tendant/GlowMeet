import { useState, useContext, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import UserContext from '../context/UserContext';
import './UserDetails.css'; // Reusing UserDetails styles for consistency

const Profile = () => {
    const navigate = useNavigate();
    const user = useContext(UserContext);
    const [interests, setInterests] = useState('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');

    // Initialize state from existing user data if available
    useEffect(() => {
        if (user && user.interests) {
            setInterests(user.interests);
        }
    }, [user]);

    const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

    const handleSubmit = async (e) => {
        e.preventDefault();
        setLoading(true);
        setError('');
        setSuccess('');

        if (interests.length > 512) {
            setError('Interests cannot exceed 512 characters.');
            setLoading(false);
            return;
        }

        try {
            const response = await fetch(`${apiBase}/api/me`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    interests: interests,

                }),
            });

            if (response.ok) {
                setSuccess('Profile updated successfully!');
                // Ideally, we should also trigger a refresh of the user context
                setTimeout(() => navigate('/'), 1500); // Redirect after short delay
            } else {
                const data = await response.json();
                setError(data.error || 'Failed to update profile.');
            }
        } catch (err) {
            console.error(err);
            setError('An error occurred. Please try again.');
        } finally {
            setLoading(false);
        }
    };

    if (!user) {
        return <div className="user-details-container center-content">Loading...</div>;
    }

    return (
        <div className="user-details-container">
            <header className="details-header">
                <button onClick={() => navigate('/')} className="back-btn">
                    ← Back to Dashboard
                </button>
            </header>

            <main className="details-content">
                <div className="ai-bg-card" style={{ maxWidth: '800px', margin: '0 auto' }}>
                    <h1 className="details-username-hero" style={{ position: 'relative', transform: 'none', top: 'auto', left: 'auto', padding: '40px 0', fontSize: '2.5rem' }}>
                        Edit Profile
                    </h1>

                    <div style={{ padding: '0 30px 40px 30px' }}>
                        <div className="profile-section">
                            <div className="user-info-grid" style={{ display: 'flex', gap: '20px', marginBottom: '30px', alignItems: 'center' }}>
                                <img
                                    src={user.profile_image_url}
                                    alt="Profile"
                                    style={{ width: '100px', height: '100px', borderRadius: '50%', objectFit: 'cover', border: '2px solid rgba(255,255,255,0.2)' }}
                                />
                                <div>
                                    <h2 style={{ margin: '0 0 5px 0', fontSize: '1.5rem' }}>{user.name}</h2>
                                    <p style={{ margin: 0, color: '#aaa' }}>{user.username}</p>
                                </div>
                            </div>

                            <form onSubmit={handleSubmit} className="edit-profile-form">
                                <div className="form-group" style={{ marginBottom: '20px' }}>
                                    <label htmlFor="interests" style={{ display: 'block', marginBottom: '8px', color: '#ccc', fontWeight: 'bold' }}>
                                        Interests <span style={{ fontSize: '0.8rem', fontWeight: 'normal', color: '#666' }}>(Max 512 chars)</span>
                                    </label>
                                    <textarea
                                        id="interests"
                                        value={interests}
                                        onChange={(e) => setInterests(e.target.value)}
                                        rows="6"
                                        placeholder="What are you interested in? (e.g., AI, Hiking, Coffee...)"
                                        style={{
                                            width: '100%',
                                            padding: '12px',
                                            borderRadius: '8px',
                                            background: 'rgba(255,255,255,0.05)',
                                            border: '1px solid rgba(255,255,255,0.1)',
                                            color: 'white',
                                            fontSize: '1rem',
                                            fontFamily: 'inherit',
                                            resize: 'vertical'
                                        }}
                                        maxLength={512}
                                    />
                                    <div style={{ textAlign: 'right', fontSize: '0.8rem', color: interests.length > 500 ? 'orange' : '#666', marginTop: '4px' }}>
                                        {interests.length}/512
                                    </div>
                                </div>

                                {error && <div className="error-message" style={{ color: '#ff6b6b', marginBottom: '15px' }}>{error}</div>}
                                {success && <div className="success-message" style={{ color: '#51cf66', marginBottom: '15px' }}>{success}</div>}

                                <button
                                    type="submit"
                                    disabled={loading}
                                    className="orb-cta"
                                    style={{
                                        width: '100%',
                                        padding: '12px',
                                        borderRadius: '8px',
                                        background: 'linear-gradient(135deg, #FFB366 0%, #FF8C00 100%)',
                                        border: 'none',
                                        cursor: loading ? 'not-allowed' : 'pointer',
                                        opacity: loading ? 0.7 : 1,
                                        fontSize: '1.1rem',
                                        marginTop: '10px'
                                    }}
                                >
                                    {loading ? 'Saving...' : 'Save Profile'}
                                </button>
                            </form>
                        </div>
                    </div>
                </div>
            </main>
        </div>
    );
};

export default Profile;
