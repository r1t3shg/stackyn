# Stackyn Frontend

A React frontend application built with Vite for managing applications and deployments on the Stackyn PaaS platform.

## Features

- 📱 **App Management**: Create, view, and delete applications
- 🚀 **Deployment Tracking**: Monitor deployment status and view logs
- 🎨 **Modern UI**: Beautiful, responsive interface built with Tailwind CSS
- ⚡ **Fast Development**: Lightning-fast HMR with Vite
- 🔄 **Client-Side Routing**: React Router for seamless navigation

## Getting Started

### Prerequisites

- Node.js 20+ and npm
- Backend API server running (see backend README)

### Installation

1. Install dependencies:
```bash
npm install
```

2. Configure the API base URL:
   - Copy `.env.example` to `.env.local`
   - Update `VITE_API_BASE_URL` to match your backend server URL

3. Run the development server:
```bash
npm run dev
```

4. Open [http://localhost:3000](http://localhost:3000) in your browser

## Project Structure

```
frontend/
├── src/
│   ├── pages/              # Page components
│   │   ├── Home.tsx        # Apps list page
│   │   ├── NewApp.tsx      # Create new app
│   │   ├── AppDetails.tsx  # App details page
│   │   └── DeploymentDetails.tsx # Deployment details
│   ├── components/         # React components
│   │   ├── AppCard.tsx
│   │   ├── DeploymentCard.tsx
│   │   ├── StatusBadge.tsx
│   │   └── LogsViewer.tsx
│   ├── lib/                # Utility functions
│   │   ├── api.ts         # API client functions
│   │   ├── config.ts      # Configuration
│   │   └── types.ts       # TypeScript type definitions
│   ├── App.tsx            # Main app component with routing
│   ├── main.tsx           # Entry point
│   └── index.css          # Global styles
├── public/                # Static assets
├── index.html             # HTML template
└── vite.config.ts         # Vite configuration
```

## API Integration

The frontend communicates with the backend API at the following endpoints:

- `GET /api/v1/apps` - List all apps
- `POST /api/v1/apps` - Create a new app
- `GET /api/v1/apps/{id}` - Get app by ID
- `DELETE /api/v1/apps/{id}` - Delete an app
- `POST /api/v1/apps/{id}/redeploy` - Redeploy an app
- `GET /api/v1/apps/{id}/deployments` - List deployments for an app
- `GET /api/v1/deployments/{id}` - Get deployment by ID
- `GET /api/v1/deployments/{id}/logs` - Get deployment logs

## Building for Production

```bash
npm run build
```

This creates a `dist/` directory with optimized production files. Serve these files with a static file server like nginx.

For preview:
```bash
npm run preview
```

## Environment Variables

- `VITE_API_BASE_URL`: Base URL for the backend API (default: `http://localhost:8080`)

**Note**: In Vite, only environment variables prefixed with `VITE_` are exposed to the client code.

## Deployment

See [DEPLOYMENT.md](./DEPLOYMENT.md) for detailed deployment instructions.

## Development

- **Dev Server**: `npm run dev` - Starts Vite dev server with HMR
- **Build**: `npm run build` - Creates production build
- **Preview**: `npm run preview` - Preview production build locally
- **Lint**: `npm run lint` - Run ESLint

## Technologies

- **React 19** - UI library
- **Vite 6** - Build tool and dev server
- **React Router 7** - Client-side routing
- **TypeScript** - Type safety
- **Tailwind CSS 4** - Styling
