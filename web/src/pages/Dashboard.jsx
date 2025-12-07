import { useState, useContext } from "react";
import UserContext from "../context/UserContext";
import Header from "../components/Header";
import "./Dashboard.css";

const Dashboard = () => {
    const user = useContext(UserContext);
    const [isActive, setIsActive] = useState(false);

    const toggleConnection = () => {
        setIsActive(!isActive);
    };

    return (
        <div className="app dashboard-container">
            <Header user={user} showLogout={true} />

            <div className="dashboard-content">
                <div className="user-display">
                    <h1 className="large-username">{user?.name || user?.username || "Creator"}</h1>
                    <p className="connection-status">
                        {isActive ? "searching for connections..." : "currently unconnected"}
                    </p>
                </div>

                <div
                    className={`connection-orb ${isActive ? 'active' : ''}`}
                    onClick={toggleConnection}
                >
                    <span className="orb-cta">
                        {isActive ? "Searching..." : "Connect Now"}
                    </span>
                </div>
            </div>

            <div className="glow-orb" style={{ top: '50%', width: '800px', height: '800px', opacity: 0.2 }}></div>
        </div>
    );
};

export default Dashboard;
