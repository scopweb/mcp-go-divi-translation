// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  integrations: [
    starlight({
      title: 'Divi Translator MCP',
      description: 'Automatic Divi page translation for Claude Desktop — preserves all [et_*] shortcodes and HTML structure.',
      logo: {
        src: './src/assets/scopweb.png',
        alt: 'scopweb',
      },
      // Dark + light themes (dark by default)
      expressiveCode: {
        themes: ['starlight-dark', 'starlight-light'],
      },
      // Google Fonts via head array (Starlight-native approach)
      head: [
        {
          tag: 'link',
          attrs: { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        },
        {
          tag: 'link',
          attrs: { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: true },
        },
        {
          tag: 'link',
          attrs: {
            rel: 'stylesheet',
            href: 'https://fonts.googleapis.com/css2?family=DM+Sans:ital,opsz,wght@0,9..40,300;0,9..40,400;0,9..40,500;0,9..40,600;1,9..40,400&family=Space+Mono:ital,wght@0,400;0,700;1,400&display=swap',
          },
        },
        {
          tag: 'script',
          content: `if (!localStorage.getItem('starlight-theme')) {
  document.documentElement.dataset.theme = 'dark';
  localStorage.setItem('starlight-theme', 'dark');
}`,
        },
      ],
      // Custom CSS — scopweb theme
      customCss: ['./src/styles/custom.css'],
      // Sidebar navigation
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Overview', link: '/' },
            { label: 'Installation', link: '/installation/' },
          ],
        },
        {
          label: 'Usage',
          items: [
            { label: 'Tools Reference', link: '/tools/' },
            { label: 'WordPress Integration', link: '/wordpress/' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'Translation Rules', link: '/translation-rules/' },
            { label: 'Security & Configuration', link: '/security/' },
          ],
        },
      ],
      // Social links
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/scopweb/mcp-go-divi-translation',
        },
        {
          icon: 'external',
          label: 'scopweb.com',
          href: 'https://scopweb.com',
        },
      ],
    }),
  ],
});
