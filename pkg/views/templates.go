package views

// layoutHeader is the opening portion of the HTML layout.
const layoutHeader = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Omnidex - Documentation Portal</title>
    <script src="/static/js/htmx.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/mermaid@11.12.3/dist/mermaid.min.js"></script>
    <link rel="stylesheet" href="/static/css/style.css">
    <script>
        if (typeof mermaid !== 'undefined') { mermaid.initialize({startOnLoad: true}); }
        document.addEventListener('htmx:afterSwap', function(event) {
            if (typeof mermaid !== 'undefined') {
                var target = event.detail.elt;
                var nodes = target.querySelectorAll('.mermaid:not([data-processed])');
                if (nodes.length > 0) { mermaid.run({nodes: Array.from(nodes)}); }
            }
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
    <article class="flex-1 min-w-0">
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
            <h3 class="text-lg font-medium text-blue-600 mb-1">{{.Title}}</h3>
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
            <h2 class="text-lg font-medium text-blue-600 mb-1">{{.Title}}</h2>
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
