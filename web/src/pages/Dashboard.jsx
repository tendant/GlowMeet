import { useState, useContext } from "react";
import { useNavigate } from "react-router-dom";
import UserContext from "../context/UserContext";
import "./Dashboard.css";

const Dashboard = () => {
    const navigate = useNavigate();
    const user = useContext(UserContext);
    const [isActive, setIsActive] = useState(false);

    const handleLogout = () => {
        // Clear tokens/state if stored client side
        document.cookie = "access_token=; Max-Age=0; path=/";
        document.cookie = "refresh_token=; Max-Age=0; path=/";
        navigate("/login");
    };

    const toggleConnection = () => {
        setIsActive(!isActive);
    };

    return (
        <div className="app">
            <header className="header" style={{ position: "relative" }}>
                <div className="container header-content">
                    <div className="logo glow-text">GlowMeet</div>
                    <nav className="nav">
                        <button className="btn-ghost" onClick={handleLogout}>
                            Logout
                        </button>
                    </nav>
                </div>
            </header>

            <div className="center-content">
                <div className="joy-card">
                    <h2 className="joy-title">{user?.name || user?.username || "Joy"}</h2>

                    <div
                        className={`connection-circle ${isActive ? 'active' : ''}`}
                        onClick={toggleConnection}
                    >
                        <span className="circle-text">
                            {isActive ? "Open to connect" : "Open Now!"}
                        </span>
                    </div>
                </div>
            </div>

            <div className="glow-orb" style={{ top: '20%', width: '500px', height: '500px' }}></div>
        </div>
    );
};

export default Dashboard;
