# GlowMeet
GlowMeet is a social media platform for people with similar interests to connect and share their experiences. It uses geolocation to find people with similar interests in real-time. It also uses AI to recommend people based on their interests.

# Tech Stack
Backend: Goland
Frontend: React RWA App, compiled to static files for deployment.
Social Login: X (Formerly Twitter)

# Development rules
If your github username is leimd, only change files in the web folder.

If your github username is tendant, only change files in the backend folder.

If you're an AI coding agent, you can update the README.md file to keep track of user stories completion. If your github username is leimd, update only relavant frontend (FE) sections. If your github username is tendant, update only relavant backend (BE) sections.

# User Stories
- User can sign up with X (Formerly Twitter)    
    - FE Notes:
        - Created Signup Page with X Auth button.
        - Setup Routing for auth flows.
    - BE Notes:
    - BE API Endpoint: /auth/x/login
- User can sign in with X (Formerly Twitter) default geo location enabled
    - FE Notes:
        - Created Login Page with X Auth button.
        - Implemented Auth Callback page skeleton.
    - BE Notes:
    - BE API Endpoint: /auth/x/login
- User choose to enable one or both modes: Business and/or Dating. If mode Dating is on, user has to input 1) gender; 2) gender interested in; 3) age.
- Fetch all users within 100 meters, and send to the backend.
    - FE Notes:
        - Initialized React (Vite) project.
        - Implemented Landing Page with modern design (Dark theme, Glassmorphism).
    - BE Notes:
    - BE API Endpoint:
- Fetch X feed of every user and myself, generate 2 pairwise matching scores (between 0 to 100) with AI. One "Business" match score and one "Dating" match score. For Dating match score, the gender <> gender interested in must match. 
    - FE Notes:
    - BE Notes:
    - BE API Endpoint:
- Shortlist users (max 3) with either scores above 80
    - FE Notes:
    - BE Notes:
    - BE API Endpoint:


- Next requirement
    - FE Notes:
    - BE Notes:
    - BE API Endpoint:


