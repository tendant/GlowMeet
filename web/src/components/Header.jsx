import { useNavigate } from "react-router-dom";
import './Header.css';

const Header = ({ user, showLogout = true }) => {
    const navigate = useNavigate();

    const handleLogout = () => {
        document.cookie = "access_token=; Max-Age=0; path=/";
        document.cookie = "refresh_token=; Max-Age=0; path=/";
        navigate("/login");
    };

    return (
        <div className="app-header">
            <div className="header-left">
                <div className="logo glow-text">GlowMeet</div>
                {user && (
                    <div
                        className="user-greeting"
                        onClick={() => navigate('/profile')}
                        style={{ cursor: 'pointer' }}
                        title="Edit Profile"
                    >
                        Hello, {user.name || user.username || "Creator"}
                    </div>
                )}
            </div>
            {showLogout && (
                <nav className="header-nav">
                    <button className="btn-ghost" onClick={handleLogout}>
                        Logout
                    </button>
                </nav>
            )}
        </div>
    );
};

export default Header;
