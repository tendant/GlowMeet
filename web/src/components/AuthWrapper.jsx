import { useEffect, useState } from "react";
import { Outlet, useNavigate } from "react-router-dom";
import UserContext from "../context/UserContext";

const AuthWrapper = () => {
    const navigate = useNavigate();
    const [loading, setLoading] = useState(true);
    const [user, setUser] = useState(null);
    const apiBase = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');

    useEffect(() => {
        const checkAuth = async () => {
            try {
                const response = await fetch(`${apiBase}/api/me`);
                if (!response.ok) {
                    throw new Error("Unauthorized");
                }
                const userData = await response.json();
                setUser(userData);
                setLoading(false);
            } catch (error) {
                console.error("Auth check failed:", error);
                navigate("/login");
            }
        };
        checkAuth();
    }, [navigate, apiBase]);

    useEffect(() => {
        if (loading) return;

        const sendLocation = () => {
            if (!navigator.geolocation) return;

            navigator.geolocation.getCurrentPosition(
                async (position) => {
                    try {
                        const { latitude, longitude } = position.coords;
                        await fetch(`${apiBase}/api/me/location`, {
                            method: 'POST',
                            headers: {
                                'Content-Type': 'application/json',
                            },
                            body: JSON.stringify({ lat: latitude, long: longitude })
                        });
                        console.log(`[Geo] Sent location: ${latitude}, ${longitude}`);
                    } catch (err) {
                        console.error("[Geo] Failed to update location:", err);
                    }
                },
                (err) => console.error("[Geo] Error getting location:", err),
                { enableHighAccuracy: true }
            );
        };

        // Send immediately, then every 10s
        sendLocation();
        const interval = setInterval(sendLocation, 10000);
        return () => clearInterval(interval);
    }, [loading, apiBase]);

    if (loading) {
        return (
            <div className="app center-content">
                <div className="glow-orb" style={{ width: '200px', height: '200px' }}></div>
                <p className="glow-text">Loading...</p>
            </div>
        );
    }

    return (
        <UserContext.Provider value={user}>
            <Outlet />
        </UserContext.Provider>
    );
};

export default AuthWrapper;
