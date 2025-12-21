# Migration from Next.js to React (Vite)

This document summarizes the migration from Next.js to React with Vite.

## Changes Made

### 1. Build System
- **Before**: Next.js 16
- **After**: Vite 6 with React 19

### 2. Routing
- **Before**: Next.js file-based routing (`app/` directory)
- **After**: React Router 7 with explicit route definitions

### 3. Environment Variables
- **Before**: `NEXT_PUBLIC_API_BASE_URL`
- **After**: `VITE_API_BASE_URL`

### 4. Project Structure
- **Before**: 
  ```
  app/
  components/
  lib/
  ```
- **After**:
  ```
  src/
    pages/
    components/
    lib/
  ```

### 5. Development Server
- **Before**: `next dev` (port 3000)
- **After**: `vite` (port 3000)

### 6. Build Output
- **Before**: `.next/` directory (server-side rendering)
- **After**: `dist/` directory (static files)

### 7. Production Deployment
- **Before**: Node.js server with `next start`
- **After**: Static file server (nginx) serving `dist/` directory

### 8. Docker
- **Before**: Node.js runtime with Next.js server
- **After**: Nginx serving static files

## Key Differences

1. **No Server-Side Rendering**: Vite builds a pure client-side React app
2. **Static Files**: Production build creates static HTML/CSS/JS files
3. **Environment Variables**: Must be prefixed with `VITE_` to be exposed
4. **Routing**: Uses React Router instead of file-based routing
5. **Imports**: Changed from Next.js imports (`next/link`, `next/navigation`) to React Router (`react-router-dom`)

## Migration Checklist

- [x] Update package.json with Vite dependencies
- [x] Create vite.config.ts
- [x] Create index.html
- [x] Move pages to src/pages/
- [x] Move components to src/components/
- [x] Move lib files to src/lib/
- [x] Update all imports (remove Next.js, add React Router)
- [x] Update environment variable usage
- [x] Update Dockerfile for static file serving
- [x] Update deployment documentation
- [x] Remove Next.js specific files

## Old Files to Remove (Optional)

The following directories/files are no longer needed but kept for reference:
- `app/` - Old Next.js pages
- `components/` (root) - Old components (now in src/components/)
- `lib/` (root) - Old lib files (now in src/lib/)

You can safely delete these after verifying the new app works correctly.

## Testing

1. Install dependencies: `npm install`
2. Run dev server: `npm run dev`
3. Build for production: `npm run build`
4. Preview build: `npm run preview`

## Notes

- The API integration remains the same
- All components and pages have been converted
- Styling (Tailwind CSS) remains unchanged
- TypeScript configuration updated for Vite

