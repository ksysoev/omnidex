package views

// layoutHeader is the opening portion of the HTML layout.
const layoutHeader = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Omnidex - Documentation Portal</title>
    <script src="/static/js/htmx.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/mermaid@11.12.3/dist/mermaid.min.js" integrity="sha384-jFhLSLFn4m565eRAS0CDMWubMqOtfZWWbE8kqgGdU+VHbJ3B2G/4X8u+0BM8MtdU" crossorigin="anonymous"></script>
    <link rel="stylesheet" href="/static/css/style.css">
    <script>
        if (typeof mermaid !== 'undefined') {
            mermaid.initialize({
                startOnLoad: true,
                theme: 'base',
                themeVariables: {
                    background: '#f9fafb',
                    fontFamily: 'ui-sans-serif, system-ui, sans-serif',
                    primaryColor: '#eff6ff',
                    primaryBorderColor: '#93c5fd',
                    primaryTextColor: '#1e3a5f',
                    secondaryColor: '#f3f4f6',
                    secondaryBorderColor: '#d1d5db',
                    tertiaryColor: '#f9fafb',
                    tertiaryBorderColor: '#e5e7eb',
                    lineColor: '#9ca3af',
                    textColor: '#374151',
                    noteBkgColor: '#eff6ff',
                    noteBorderColor: '#93c5fd',
                    actorBkg: '#ffffff',
                    actorBorder: '#d1d5db'
                }
            });
        }
        function initScrollSpy() {
            if (window._tocObserver) {
                window._tocObserver.disconnect();
                window._tocObserver = null;
            }
            window._tocActiveId = null;
            window._tocHeadingStates = {};
            if (!('IntersectionObserver' in window)) return;
            var links = document.querySelectorAll('[data-toc-link]');
            if (!links.length) return;
            var content = document.getElementById('doc-content');
            if (!content) return;
            var headings = content.querySelectorAll('.prose h1[id], .prose h2[id], .prose h3[id]');
            if (!headings.length) return;
            window._tocObserver = new IntersectionObserver(function(entries) {
                entries.forEach(function(entry) {
                    if (entry.target.id) {
                        window._tocHeadingStates[entry.target.id] = entry.isIntersecting;
                    }
                });
                var activeId = null;
                for (var i = 0; i < headings.length; i++) {
                    if (window._tocHeadingStates[headings[i].id]) {
                        activeId = headings[i].id;
                        break;
                    }
                }
                if (!activeId || window._tocActiveId === activeId) return;
                window._tocActiveId = activeId;
                links.forEach(function(l) { l.classList.remove('toc-active'); });
                var escapedId = (window.CSS && window.CSS.escape) ? window.CSS.escape(activeId) : activeId;
                var active = document.querySelector('[data-toc-link="' + escapedId + '"]');
                if (active) { active.classList.add('toc-active'); }
            }, { rootMargin: '0px 0px -80% 0px', threshold: 0 });
            headings.forEach(function(h) {
                window._tocObserver.observe(h);
            });
            var hash = window.location.hash;
            if (hash && hash.charAt(0) === '#') {
                var id = hash.slice(1);
                try { id = decodeURIComponent(id); } catch (e) { /* use raw id */ }
                var target = document.getElementById(id);
                if (target) {
                    var scrollBehavior = 'smooth';
                    if (window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
                        scrollBehavior = 'auto';
                    }
                    target.scrollIntoView({behavior: scrollBehavior});
                }
            }
        }
        document.addEventListener('DOMContentLoaded', function() { initScrollSpy(); });
        document.addEventListener('htmx:afterSwap', function(event) {
            if (typeof mermaid !== 'undefined') {
                var target = event.detail.elt;
                var nodes = target.querySelectorAll('.mermaid:not([data-processed])');
                if (nodes.length > 0) { mermaid.run({nodes: Array.from(nodes)}).catch(function(e) { console.error('Mermaid rendering failed:', e); }); }
            }
            initScrollSpy();
        });
    </script>
</head>
<body class="bg-gray-50 min-h-screen flex flex-col">
    <nav class="bg-white border-b border-gray-200 px-6 py-3">
        <div class="max-w-7xl mx-auto flex items-center justify-between">
            <a href="/" class="text-xl font-bold text-gray-900" hx-get="/" hx-target="#main-content" hx-push-url="true">
                Omnidex
            </a>
            <div class="flex items-center gap-4">
                <input type="search" name="q" placeholder="Search documentation..."
                    class="w-64 px-4 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    hx-get="/search" hx-trigger="keyup changed delay:300ms" hx-target="#main-content" hx-push-url="true">
            </div>
        </div>
    </nav>
    <main id="main-content" class="max-w-7xl mx-auto px-6 py-8 flex-1 w-full">`

// layoutFooter is the closing portion of the HTML layout.
const layoutFooter = `</main>
    <footer class="border-t border-gray-200 py-6 text-center text-sm text-gray-500">
        <p>Powered by Omnidex</p>
    </footer>
</body>
</html>`

// homeContentBody is the home page content template.
const homeContentBody = `
<div>
    <h1 class="text-3xl font-bold text-gray-900 mb-6">Documentation Portal</h1>
    {{if .Repos}}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {{range .Repos}}
        <a href="/docs/{{.Name}}/"
           hx-get="/docs/{{.Name}}/" hx-target="#main-content" hx-push-url="true"
           class="block p-6 bg-white rounded-lg border border-gray-200 hover:border-blue-500 hover:shadow-md transition-all">
            <h2 class="text-lg font-semibold text-gray-900 mb-2">{{.Name}}</h2>
            <div class="flex items-center gap-4 text-sm text-gray-500">
                <span>{{.DocCount}} documents</span>
                <span>Updated {{.LastUpdated.Format "Jan 02, 2006"}}</span>
            </div>
        </a>
        {{end}}
    </div>
    {{else}}
    <div class="text-center py-16">
        <p class="text-gray-500 text-lg mb-4">No repositories indexed yet.</p>
        <p class="text-gray-400">Configure the Omnidex GitHub Action in your repositories to get started.</p>
    </div>
    {{end}}
</div>`

// docContentBody is the document page content template.
const docContentBody = `
<div class="flex gap-8">
    <aside class="w-64 flex-shrink-0 hidden md:block">
        <nav class="sticky top-8">
            <h3 class="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">{{.Doc.Repo}}</h3>
            <ul class="space-y-1">
                {{range .NavDocs}}
                <li>
                    <a href="/docs/{{.Repo}}/{{.Path}}"
                       hx-get="/docs/{{.Repo}}/{{.Path}}" hx-target="#main-content" hx-push-url="true"
                       class="block px-3 py-1.5 text-sm rounded-md hover:bg-gray-100 text-gray-700 hover:text-gray-900">
                        {{.Title}}
                    </a>
                </li>
                {{end}}
            </ul>
        </nav>
    </aside>
    <article id="doc-content" class="flex-1 min-w-0">
        <div class="mb-4 text-sm text-gray-500">
            <a href="/" hx-get="/" hx-target="#main-content" hx-push-url="true" class="hover:text-blue-600">Home</a>
            <span class="mx-1">/</span>
            <a href="/docs/{{.Doc.Repo}}/" hx-get="/docs/{{.Doc.Repo}}/" hx-target="#main-content" hx-push-url="true" class="hover:text-blue-600">{{.Doc.Repo}}</a>
            <span class="mx-1">/</span>
            <span>{{.Doc.Path}}</span>
        </div>
        <div class="prose prose-gray max-w-none bg-white rounded-lg border border-gray-200 p-8">
            {{html .HTML}}
        </div>
    </article>
    {{if gt (len .Headings) 1}}
    <aside class="w-56 flex-shrink-0 hidden lg:block">
        <nav class="sticky top-8">
            <h3 class="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">On this page</h3>
            <ul class="space-y-1 border-l border-gray-200">
                {{range .Headings}}
                <li>
                    <a href="#{{.ID}}" data-toc-link="{{.ID}}"
                       class="toc-link block py-1 text-sm text-gray-500 hover:text-gray-900 border-l-2 border-transparent hover:border-gray-400 -ml-px {{tocIndent .Level}}">
                        {{.Text}}
                    </a>
                </li>
                {{end}}
            </ul>
        </nav>
    </aside>
    {{end}}
</div>`

// searchContentBody is the search page content template.
const searchContentBody = `
<div>
    <h1 class="text-3xl font-bold text-gray-900 mb-6">Search Documentation</h1>
    <div id="search-results">` + searchResultsBody + `</div>
</div>`

// searchResultsBody is the search results partial template.
const searchResultsBody = `{{if .Results}}
    <p class="text-sm text-gray-500 mb-4">{{.Results.Total}} results found</p>
    {{if .Results.Hits}}
    <div class="space-y-4">
        {{range .Results.Hits}}
        <a href="/docs/{{.Repo}}/{{.Path}}" hx-get="/docs/{{.Repo}}/{{.Path}}" hx-target="#main-content" hx-push-url="true"
           class="block p-4 bg-white rounded-lg border border-gray-200 hover:border-blue-500 hover:shadow-sm transition-all">
            <h3 class="text-lg font-semibold text-gray-900 mb-2">{{.Title}}</h3>
            <p class="text-sm text-gray-500 mb-2">{{.Repo}}/{{.Path}}</p>
            {{range .Fragments}}
            <p class="text-sm text-gray-600">{{.}}</p>
            {{end}}
        </a>
        {{end}}
    </div>
    {{else}}
    <p class="text-gray-500">No results found for &ldquo;{{$.Query}}&rdquo;.</p>
    {{end}}
{{else if .Query}}
    <p class="text-gray-500">No results found for &ldquo;{{.Query}}&rdquo;.</p>
{{else}}
    <p class="text-gray-400">Enter a search query above to find documentation.</p>
{{end}}`

// repoIndexContentBody is the repo index page content template.
const repoIndexContentBody = `
<div>
    <div class="mb-4 text-sm text-gray-500">
        <a href="/" hx-get="/" hx-target="#main-content" hx-push-url="true" class="hover:text-blue-600">Home</a>
        <span class="mx-1">/</span>
        <span>{{.Repo}}</span>
    </div>
    <h1 class="text-3xl font-bold text-gray-900 mb-6">{{.Repo}}</h1>
    {{if .Docs}}
    <div class="space-y-3">
        {{range .Docs}}
        <a href="/docs/{{.Repo}}/{{.Path}}"
           hx-get="/docs/{{.Repo}}/{{.Path}}" hx-target="#main-content" hx-push-url="true"
           class="block p-4 bg-white rounded-lg border border-gray-200 hover:border-blue-500 hover:shadow-sm transition-all">
            <h2 class="text-lg font-semibold text-gray-900 mb-2">{{.Title}}</h2>
            <div class="flex items-center gap-4 text-sm text-gray-500">
                <span>{{.Path}}</span>
                <span>Updated {{.UpdatedAt.Format "Jan 02, 2006"}}</span>
            </div>
        </a>
        {{end}}
    </div>
    {{else}}
    <div class="text-center py-16">
        <p class="text-gray-500 text-lg mb-4">No documents in this repository yet.</p>
        <p class="text-gray-400">Publish documentation using the Omnidex GitHub Action to get started.</p>
    </div>
    {{end}}
</div>`

// openapiDocContentBody is the document page template for OpenAPI specs rendered via Scalar API Reference.
// The Scalar script is loaded from CDN only when an OpenAPI document is displayed (lazy-loading).
// The spec JSON is embedded inline and fed to Scalar on initialisation.
const openapiDocContentBody = `
<div class="flex gap-8">
    <aside class="w-64 flex-shrink-0 hidden md:block">
        <nav class="sticky top-8">
            <h3 class="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">{{.Doc.Repo}}</h3>
            <ul class="space-y-1">
                {{range .NavDocs}}
                <li>
                    <a href="/docs/{{.Repo}}/{{.Path}}"
                       hx-get="/docs/{{.Repo}}/{{.Path}}" hx-target="#main-content" hx-push-url="true"
                       class="block px-3 py-1.5 text-sm rounded-md hover:bg-gray-100 text-gray-700 hover:text-gray-900">
                        {{.Title}}
                    </a>
                </li>
                {{end}}
            </ul>
        </nav>
    </aside>
    <article id="doc-content" class="flex-1 min-w-0">
        <div class="mb-4 text-sm text-gray-500">
            <a href="/" hx-get="/" hx-target="#main-content" hx-push-url="true" class="hover:text-blue-600">Home</a>
            <span class="mx-1">/</span>
            <a href="/docs/{{.Doc.Repo}}/" hx-get="/docs/{{.Doc.Repo}}/" hx-target="#main-content" hx-push-url="true" class="hover:text-blue-600">{{.Doc.Repo}}</a>
            <span class="mx-1">/</span>
            <span>{{.Doc.Path}}</span>
        </div>
        <div class="bg-white rounded-lg border border-gray-200 p-4">
            <div id="scalar-api-reference"></div>
            <script type="application/json" id="openapi-spec">{{safeJS .HTML}}</script>
            <script>
            (function() {
                var specEl = document.getElementById('openapi-spec');
                if (!specEl) return;
                var spec;
                try {
                    spec = JSON.parse(specEl.textContent);
                } catch (e) {
                    console.error('Failed to parse OpenAPI spec JSON from #openapi-spec:', e);
                    return;
                }

                function initScalar() {
                    var container = document.getElementById('scalar-api-reference');
                    if (!container) return;
                    container.innerHTML = '';
                    Scalar.createApiReference('#scalar-api-reference', {
                        content: spec,
                        theme: 'none',
                        layout: 'modern',
                        withDefaultFonts: false,
                        forceDarkModeState: 'light',
                        hideDarkModeToggle: true,
                        showSidebar: false,
                        hideSearch: true,
                        hideClientButton: true,
                        hideTestRequestButton: true,
                        telemetry: false,
                        showDeveloperTools: 'never'
                    });
                }

                if (typeof window.Scalar !== 'undefined' && typeof window.Scalar.createApiReference === 'function') {
                    initScalar();
                    return;
                }

                var existingScript = document.querySelector('script[data-scalar-api-reference]');
                if (existingScript) {
                    if (existingScript.dataset.loaded === 'true') {
                        initScalar();
                    } else {
                        existingScript.addEventListener('load', initScalar);
                    }
                    return;
                }

                var script = document.createElement('script');
                script.src = 'https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.46.0';
                script.integrity = 'sha384-J8SKUvgS9P4wa0c+HdF7IJMAxLKPA2MTTiMrMHEnBGrImueMygyFW5kWh60jyN1j';
                script.crossOrigin = 'anonymous';
                script.async = true;
                script.setAttribute('data-scalar-api-reference', 'true');
                script.onload = function() {
                    script.dataset.loaded = 'true';
                    initScalar();
                };
                document.head.appendChild(script);
            })();
            </script>
        </div>
    </article>
</div>`

// notFoundBody is the 404 page content template.
const notFoundBody = `
<div class="text-center py-16">
    <h1 class="text-4xl font-bold text-gray-900 mb-4">404 - Not Found</h1>
    <p class="text-gray-500 mb-8">The page you are looking for does not exist.</p>
    <a href="/" hx-get="/" hx-target="#main-content" hx-push-url="true"
       class="inline-block px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
        Go Home
    </a>
</div>`
