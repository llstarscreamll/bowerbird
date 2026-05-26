import re

with open("apps/backend/cmd/api/main.go", "r") as f:
    content = f.read()

is_dev_code = """
	isDev := cfg.AppEnv == "development" || cfg.AppEnv == "local"
	mux := http.NewServeMux()
	healthHandler.Register(mux, isDev)
"""

content = content.replace("""
	mux := http.NewServeMux()
	healthHandler.Register(mux)
""", is_dev_code)

content = content.replace("authHandler.Register(mux, authMiddleware)", "authHandler.Register(mux, authMiddleware, isDev)")
content = content.replace("orgHandler.Register(mux, authMiddleware)", "orgHandler.Register(mux, authMiddleware, isDev)")
content = content.replace("inboxHandler.Register(mux, authMiddleware)", "inboxHandler.Register(mux, authMiddleware, isDev)")

with open("apps/backend/cmd/api/main.go", "w") as f:
    f.write(content)
