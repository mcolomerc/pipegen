import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'PipeGen',
  description: 'A powerful CLI tool for creating and managing streaming data pipelines • v1.4.0',
  base: '/pipegen/',
  
  head: [
    ['link', { rel: 'icon', href: '/pipegen/favicon.ico' }],
    ['link', { rel: 'stylesheet', href: 'https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'PipeGen - Streaming Data Pipeline Generator' }],
    ['meta', { property: 'og:description', content: 'Create and manage streaming data pipelines using Apache Kafka and FlinkSQL with AI-powered generation and real-time monitoring.' }],
    ['meta', { property: 'og:image', content: '/pipegen/logo.png' }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
  ],

  themeConfig: {
    logo: {
      light: '/logo.png',
      dark: '/logo.png'
    },

    nav: [
      { text: 'Home', link: '/' },
      { text: 'Getting Started', link: '/getting-started' },
      { text: 'Commands', link: '/commands' },
      { text: 'Features', link: '/features' },
      { text: 'Examples', link: '/examples' },
      {
        text: 'More',
        items: [
          { text: 'Configuration', link: '/configuration' },
          { text: 'Traffic Patterns', link: '/traffic-patterns' },
          { text: 'AI Generation', link: '/ai-generation' },
          { text: 'Changelog', link: '/changelog' }
        ]
      }
    ],

    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'What is PipeGen?', link: '/introduction' },
          { text: 'Getting Started', link: '/getting-started' },
          { text: 'Installation', link: '/installation' }
        ]
      },
      {
        text: 'Core Features',
        items: [
          { text: 'Project Scaffolding', link: '/features/scaffolding' },
          { text: 'AI-Powered Generation', link: '/ai-generation' },
          { text: 'Traffic Patterns', link: '/traffic-patterns' },
          { text: 'Execution Reports', link: '/features/reports' }
        ]
      },
      {
        text: 'Commands',
        items: [
          { text: 'Overview', link: '/commands' },
          { text: 'pipegen init', link: '/commands/init' },
          { text: 'pipegen run', link: '/commands/run' },
          { text: 'pipegen deploy', link: '/commands/deploy' },
          { text: 'pipegen validate', link: '/commands/validate' }
        ]
      },
      {
        text: 'Configuration',
        items: [
          { text: 'Configuration Files', link: '/configuration' },
          { text: 'Environment Variables', link: '/configuration/environment' },
          { text: 'Cloud Setup', link: '/configuration/cloud' }
        ]
      },
      {
        text: 'Examples & Tutorials',
        items: [
          { text: 'Overview', link: '/examples' },
          { text: 'Quick Examples', link: '/examples#quick-examples' },
          { text: 'Industry Use Cases', link: '/examples#industry-use-cases' },
          { text: 'Traffic Pattern Examples', link: '/examples#traffic-pattern-examples' },
          { text: 'Testing Scenarios', link: '/examples#testing-scenarios' },
          { text: 'Integration Examples', link: '/examples#integration-examples' },
          { text: 'Advanced Use Cases', link: '/examples#advanced-use-cases' },
          { text: 'Sample Outputs', link: '/examples#sample-outputs' },
          { text: 'Common Patterns', link: '/examples#common-patterns' }
        ]
      },
      {
        text: 'Advanced Topics',
        items: [
          { text: 'Performance Tuning', link: '/advanced/performance' },
          { text: 'Troubleshooting', link: '/advanced/troubleshooting' },
          { text: 'Contributing', link: '/advanced/contributing' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/mcolomerc/pipegen' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2025 PipeGen Team'
    },

    editLink: {
      pattern: 'https://github.com/mcolomerc/pipegen/edit/main/docs-site/:path',
      text: 'Edit this page on GitHub'
    },

    search: {
      provider: 'local'
    },

    outline: {
      level: [2, 3]
    }
  },

  markdown: {
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    },
    lineNumbers: true
  }
})
