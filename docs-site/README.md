# PipeGen Documentation Site

This directory contains the VitePress-based documentation site for PipeGen.

## Development

### Prerequisites
- Node.js 18+ 
- npm

### Setup
```bash
npm install
```

### Development Server
```bash
npm run dev
```

Visit http://localhost:5173/pipegen/ to see the site.

### Build
```bash
npm run build
```

The built site will be in `.vitepress/dist/`.

### Preview Built Site
```bash
npm run preview
```

## Structure

- `index.md` - Homepage
- `getting-started.md` - Getting started guide
- `traffic-patterns.md` - Traffic patterns documentation
- `commands.md` - Commands overview
- `configuration.md` - Configuration reference
- `dashboard.md` - Dashboard documentation
- `ai-generation.md` - AI generation features
- `examples.md` - Usage examples
- `installation.md` - Installation guide
- `commands/` - Individual command documentation
- `advanced/` - Advanced topics
- `public/` - Static assets (logos, images)
- `.vitepress/` - VitePress configuration

## Deployment

The site is automatically deployed to GitHub Pages when changes are pushed to the main branch.

## Branding

The site uses the official PipeGen branding:
- Logo: `public/logo.png`
- Banner: `public/pipegen.png`
- Screenshots: `public/screenshot.png`

## Configuration

Site configuration is in `.vitepress/config.js`. This includes:
- Navigation structure
- Theme settings  
- Social links
- Search configuration
- Base URL for GitHub Pages deployment
