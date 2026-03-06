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
    <style>
        /* Chroma syntax highlighting — github-dark theme */
        /* Background */ .bg { color: #e6edf3; background-color: #0d1117; }
        /* PreWrapper */ .chroma { color: #e6edf3; background-color: #1f2937; -webkit-text-size-adjust: none; }
        /* Error */ .chroma .err { color: #f85149 }
        /* LineLink */ .chroma .lnlinks { outline: none; text-decoration: none; color: inherit }
        /* LineTableTD */ .chroma .lntd { vertical-align: top; padding: 0; margin: 0; border: 0; }
        /* LineTable */ .chroma .lntable { border-spacing: 0; padding: 0; margin: 0; border: 0; }
        /* LineHighlight */ .chroma .hl { background-color: #6e7681 }
        /* LineNumbersTable */ .chroma .lnt { white-space: pre; -webkit-user-select: none; user-select: none; margin-right: 0.4em; padding: 0 0.4em 0 0.4em; color: #737679 }
        /* LineNumbers */ .chroma .ln { white-space: pre; -webkit-user-select: none; user-select: none; margin-right: 0.4em; padding: 0 0.4em 0 0.4em; color: #6e7681 }
        /* Line */ .chroma .line { display: flex; }
        /* Keyword */ .chroma .k { color: #ff7b72 }
        /* KeywordConstant */ .chroma .kc { color: #79c0ff }
        /* KeywordDeclaration */ .chroma .kd { color: #ff7b72 }
        /* KeywordNamespace */ .chroma .kn { color: #ff7b72 }
        /* KeywordPseudo */ .chroma .kp { color: #79c0ff }
        /* KeywordReserved */ .chroma .kr { color: #ff7b72 }
        /* KeywordType */ .chroma .kt { color: #ff7b72 }
        /* NameClass */ .chroma .nc { color: #f0883e; font-weight: bold }
        /* NameConstant */ .chroma .no { color: #79c0ff; font-weight: bold }
        /* NameDecorator */ .chroma .nd { color: #d2a8ff; font-weight: bold }
        /* NameEntity */ .chroma .ni { color: #ffa657 }
        /* NameException */ .chroma .ne { color: #f0883e; font-weight: bold }
        /* NameLabel */ .chroma .nl { color: #79c0ff; font-weight: bold }
        /* NameNamespace */ .chroma .nn { color: #ff7b72 }
        /* NameProperty */ .chroma .py { color: #79c0ff }
        /* NameTag */ .chroma .nt { color: #7ee787 }
        /* NameVariable */ .chroma .nv { color: #79c0ff }
        /* NameVariableClass */ .chroma .vc { color: #79c0ff }
        /* NameVariableGlobal */ .chroma .vg { color: #79c0ff }
        /* NameVariableInstance */ .chroma .vi { color: #79c0ff }
        /* NameVariableMagic */ .chroma .vm { color: #79c0ff }
        /* NameFunction */ .chroma .nf { color: #d2a8ff; font-weight: bold }
        /* NameFunctionMagic */ .chroma .fm { color: #d2a8ff; font-weight: bold }
        /* Literal */ .chroma .l { color: #a5d6ff }
        /* LiteralDate */ .chroma .ld { color: #79c0ff }
        /* LiteralString */ .chroma .s { color: #a5d6ff }
        /* LiteralStringAffix */ .chroma .sa { color: #79c0ff }
        /* LiteralStringBacktick */ .chroma .sb { color: #a5d6ff }
        /* LiteralStringChar */ .chroma .sc { color: #a5d6ff }
        /* LiteralStringDelimiter */ .chroma .dl { color: #79c0ff }
        /* LiteralStringDoc */ .chroma .sd { color: #a5d6ff }
        /* LiteralStringDouble */ .chroma .s2 { color: #a5d6ff }
        /* LiteralStringEscape */ .chroma .se { color: #79c0ff }
        /* LiteralStringHeredoc */ .chroma .sh { color: #79c0ff }
        /* LiteralStringInterpol */ .chroma .si { color: #a5d6ff }
        /* LiteralStringOther */ .chroma .sx { color: #a5d6ff }
        /* LiteralStringRegex */ .chroma .sr { color: #79c0ff }
        /* LiteralStringSingle */ .chroma .s1 { color: #a5d6ff }
        /* LiteralStringSymbol */ .chroma .ss { color: #a5d6ff }
        /* LiteralNumber */ .chroma .m { color: #a5d6ff }
        /* LiteralNumberBin */ .chroma .mb { color: #a5d6ff }
        /* LiteralNumberFloat */ .chroma .mf { color: #a5d6ff }
        /* LiteralNumberHex */ .chroma .mh { color: #a5d6ff }
        /* LiteralNumberInteger */ .chroma .mi { color: #a5d6ff }
        /* LiteralNumberIntegerLong */ .chroma .il { color: #a5d6ff }
        /* LiteralNumberOct */ .chroma .mo { color: #a5d6ff }
        /* Operator */ .chroma .o { color: #ff7b72; font-weight: bold }
        /* OperatorWord */ .chroma .ow { color: #ff7b72; font-weight: bold }
        /* Comment */ .chroma .c { color: #8b949e; font-style: italic }
        /* CommentHashbang */ .chroma .ch { color: #8b949e; font-style: italic }
        /* CommentMultiline */ .chroma .cm { color: #8b949e; font-style: italic }
        /* CommentSingle */ .chroma .c1 { color: #8b949e; font-style: italic }
        /* CommentSpecial */ .chroma .cs { color: #8b949e; font-weight: bold; font-style: italic }
        /* CommentPreproc */ .chroma .cp { color: #8b949e; font-weight: bold; font-style: italic }
        /* CommentPreprocFile */ .chroma .cpf { color: #8b949e; font-weight: bold; font-style: italic }
        /* GenericDeleted */ .chroma .gd { color: #ffa198; background-color: #490202 }
        /* GenericEmph */ .chroma .ge { font-style: italic }
        /* GenericError */ .chroma .gr { color: #ffa198 }
        /* GenericHeading */ .chroma .gh { color: #79c0ff; font-weight: bold }
        /* GenericInserted */ .chroma .gi { color: #56d364; background-color: #0f5323 }
        /* GenericOutput */ .chroma .go { color: #8b949e }
        /* GenericPrompt */ .chroma .gp { color: #8b949e }
        /* GenericStrong */ .chroma .gs { font-weight: bold }
        /* GenericSubheading */ .chroma .gu { color: #79c0ff }
        /* GenericTraceback */ .chroma .gt { color: #ff7b72 }
        /* GenericUnderline */ .chroma .gl { text-decoration: underline }
        /* TextWhitespace */ .chroma .w { color: #6e7681 }
    </style>
    <script>
        if (typeof mermaid !== 'undefined') {
            mermaid.initialize({
                startOnLoad: false,
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
        function scrollToHash() {
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
        }
        function initHeadingAnchors() {
            var content = document.getElementById('doc-content');
            if (!content) return;
            var headings = content.querySelectorAll('.prose h1[id], .prose h2[id], .prose h3[id]');
            headings.forEach(function(h) {
                if (h.querySelector('.heading-anchor')) return;
                var id = h.id;
                var anchor = document.createElement('a');
                anchor.className = 'heading-anchor';
                anchor.href = '#' + id;
                anchor.setAttribute('aria-label', 'Copy link to section');
                anchor.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>';
                anchor.addEventListener('click', function(e) {
                    e.preventDefault();
                    var encodedId = encodeURIComponent(id);
                    var baseUrl = window.location.href.split('#')[0];
                    var url = baseUrl + '#' + encodedId;
                    var done = function() {
                        window.location.hash = encodedId;
                        anchor.classList.add('copied');
                        setTimeout(function() { anchor.classList.remove('copied'); }, 2000);
                    };
                    var fallbackCopy = function() {
                        var ta = document.createElement('textarea');
                        ta.value = url;
                        ta.style.position = 'fixed';
                        ta.style.opacity = '0';
                        document.body.appendChild(ta);
                        ta.select();
                        try {
                            if (document.execCommand('copy')) {
                                done();
                            } else {
                                window.location.hash = encodedId;
                            }
                        } catch(ex) { window.location.hash = encodedId; }
                        document.body.removeChild(ta);
                    };
                    if (navigator.clipboard && navigator.clipboard.writeText) {
                        navigator.clipboard.writeText(url).then(done).catch(function() {
                            fallbackCopy();
                        });
                    } else {
                        fallbackCopy();
                    }
                });
                h.appendChild(anchor);
            });
        }
        document.addEventListener('DOMContentLoaded', function() {
            initScrollSpy(); scrollToHash(); initHeadingAnchors();
            if (typeof mermaid !== 'undefined') {
                mermaid.run().then(initMermaidExpand).catch(function(e) {
                    console.error('Mermaid rendering failed:', e);
                    initMermaidExpand();
                });
            }
            initImageExpand();
        });
        document.addEventListener('htmx:afterSwap', function(event) {
            initScrollSpy();
            scrollToHash();
            initHeadingAnchors();
            if (typeof mermaid !== 'undefined') {
                var target = event.detail.elt;
                var nodes = target.querySelectorAll('.mermaid:not([data-processed])');
                if (nodes.length > 0) {
                    mermaid.run({nodes: Array.from(nodes)})
                        .then(initMermaidExpand)
                        .catch(function(e) { console.error('Mermaid rendering failed:', e); initMermaidExpand(); });
                } else {
                    initMermaidExpand();
                }
            } else {
                initMermaidExpand();
            }
            initImageExpand();
        });
        document.addEventListener('htmx:beforeSwap', function() { closeMediaModal(); });

        /* ================================================================
           Media fullscreen viewer (mermaid diagrams + images)
           ================================================================ */
        (function() {
            var modal, viewport, canvas, zoomLabel;
            var scale = 1, tx = 0, ty = 0;
            var minScale = 0.05, maxScale = 20;
            var isPanning = false, hasDragged = false, panStartX = 0, panStartY = 0, panStartTx = 0, panStartTy = 0;
            var pinchStartDist = 0, pinchStartScale = 1, pinchStartTx = 0, pinchStartTy = 0;
            var modalOpen = false;
            var _boundMouseMove, _boundMouseUp, _boundWheel, _boundKeyDown, _boundTouchMove, _boundTouchEnd;
            // Focus management
            var _previousFocus = null;
            var _prevBodyOverflow = '';
            // Active element move-in/out tracking
            var _activeSvg = null, _activeSvgParent = null;
            var _activeSvgOrigWidth = null, _activeSvgOrigHeight = null, _activeSvgOrigStyle = null;
            var _activeSvgPlaceholder = null;

            function getModal() {
                if (!modal) {
                    modal    = document.getElementById('media-modal');
                    viewport = document.getElementById('media-modal-viewport');
                    canvas   = document.getElementById('media-modal-canvas');
                    zoomLabel = document.getElementById('media-zoom-level');
                    var closeBtn  = document.getElementById('media-modal-close');
                    var zoomIn    = document.getElementById('media-zoom-in');
                    var zoomOut   = document.getElementById('media-zoom-out');
                    var zoomReset = document.getElementById('media-zoom-reset');
                    if (closeBtn)  closeBtn.addEventListener('click', closeMediaModal);
                    if (zoomIn)    zoomIn.addEventListener('click', function() { applyZoom(1.25, viewport.clientWidth / 2, viewport.clientHeight / 2); });
                    if (zoomOut)   zoomOut.addEventListener('click', function() { applyZoom(0.8, viewport.clientWidth / 2, viewport.clientHeight / 2); });
                    if (zoomReset) zoomReset.addEventListener('click', fitToScreen);
                    if (modal) {
                        modal.addEventListener('click', function(e) {
                            if (hasDragged) { hasDragged = false; return; }
                            if (e.target === modal || e.target === viewport) { closeMediaModal(); }
                        });
                    }
                }
                return !!modal;
            }

            function applyTransform() {
                if (!canvas) return;
                canvas.style.transform = 'translate(' + tx + 'px, ' + ty + 'px) scale(' + scale + ')';
                if (zoomLabel) { zoomLabel.textContent = Math.round(scale * 100) + '%'; }
            }

            function applyZoom(factor, cx, cy) {
                var newScale = Math.min(maxScale, Math.max(minScale, scale * factor));
                var ratio = newScale / scale;
                tx = cx - ratio * (cx - tx);
                ty = cy - ratio * (cy - ty);
                scale = newScale;
                applyTransform();
            }

            function fitToScreen() {
                if (!canvas || !viewport) return;
                var el = canvas.querySelector('svg') || canvas.querySelector('img');
                if (!el) return;
                var vw = viewport.clientWidth  - 64;
                var vh = viewport.clientHeight - 64;
                var sw, sh;
                if (el.tagName.toLowerCase() === 'svg') {
                    // Read explicit px dimensions stamped by openMediaModal.
                    sw = parseFloat(el.getAttribute('width'))  || 0;
                    sh = parseFloat(el.getAttribute('height')) || 0;
                } else {
                    // For <img> use natural (decoded) dimensions.
                    sw = el.naturalWidth  || 0;
                    sh = el.naturalHeight || 0;
                }
                if (!sw || !sh) {
                    var br = el.getBoundingClientRect();
                    sw = br.width  || vw;
                    sh = br.height || vh;
                }
                var fitScale = Math.min(vw / sw, vh / sh);
                scale = Math.min(maxScale, Math.max(minScale, fitScale));
                tx = (viewport.clientWidth  - sw * scale) / 2;
                ty = (viewport.clientHeight - sh * scale) / 2;
                applyTransform();
            }

            function onMouseDown(e) {
                if (e.button !== 0) return;
                isPanning = true;
                hasDragged = false;
                panStartX = e.clientX; panStartY = e.clientY;
                panStartTx = tx; panStartTy = ty;
                viewport.classList.add('is-panning');
                e.preventDefault();
            }
            function onMouseMove(e) {
                if (!isPanning) return;
                var dx = e.clientX - panStartX;
                var dy = e.clientY - panStartY;
                if (!hasDragged && (Math.abs(dx) > 4 || Math.abs(dy) > 4)) { hasDragged = true; }
                tx = panStartTx + dx;
                ty = panStartTy + dy;
                applyTransform();
            }
            function onMouseUp() {
                if (!isPanning) return;
                isPanning = false;
                if (viewport) viewport.classList.remove('is-panning');
            }
            function onWheel(e) {
                e.preventDefault();
                var rect = viewport.getBoundingClientRect();
                var cx = e.clientX - rect.left;
                var cy = e.clientY - rect.top;
                var delta = e.deltaY < 0 ? 1.03 : (1 / 1.03);
                applyZoom(delta, cx, cy);
            }
            function onKeyDown(e) {
                if (!modalOpen) return;
                switch (e.key) {
                    case 'Escape': closeMediaModal(); break;
                    case 'Tab': {
                        // Focus trap: keep Tab/Shift+Tab inside the modal.
                        var focusable = modal.querySelectorAll(
                            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
                        );
                        var focusArr = Array.prototype.slice.call(focusable).filter(function(el) {
                            return !el.disabled && el.offsetParent !== null;
                        });
                        if (focusArr.length === 0) { e.preventDefault(); break; }
                        var first = focusArr[0];
                        var last  = focusArr[focusArr.length - 1];
                        if (e.shiftKey) {
                            if (document.activeElement === first) { e.preventDefault(); last.focus(); }
                        } else {
                            if (document.activeElement === last)  { e.preventDefault(); first.focus(); }
                        }
                        break;
                    }
                    case '+': case '=': e.preventDefault(); applyZoom(1.25, viewport.clientWidth / 2, viewport.clientHeight / 2); break;
                    case '-': e.preventDefault(); applyZoom(0.8, viewport.clientWidth / 2, viewport.clientHeight / 2); break;
                    case '0': e.preventDefault(); fitToScreen(); break;
                    case 'ArrowLeft':  e.preventDefault(); tx -= 40; applyTransform(); break;
                    case 'ArrowRight': e.preventDefault(); tx += 40; applyTransform(); break;
                    case 'ArrowUp':    e.preventDefault(); ty -= 40; applyTransform(); break;
                    case 'ArrowDown':  e.preventDefault(); ty += 40; applyTransform(); break;
                }
            }
            function getTouchDist(touches) {
                var dx = touches[0].clientX - touches[1].clientX;
                var dy = touches[0].clientY - touches[1].clientY;
                return Math.sqrt(dx * dx + dy * dy);
            }
            function onTouchStart(e) {
                if (e.touches.length === 1) {
                    isPanning = true;
                    panStartX = e.touches[0].clientX; panStartY = e.touches[0].clientY;
                    panStartTx = tx; panStartTy = ty;
                } else if (e.touches.length === 2) {
                    isPanning = false;
                    pinchStartDist  = getTouchDist(e.touches);
                    pinchStartScale = scale;
                    pinchStartTx = tx; pinchStartTy = ty;
                }
                e.preventDefault();
            }
            function onTouchMove(e) {
                if (e.touches.length === 1 && isPanning) {
                    tx = panStartTx + (e.touches[0].clientX - panStartX);
                    ty = panStartTy + (e.touches[0].clientY - panStartY);
                    applyTransform();
                } else if (e.touches.length === 2) {
                    var dist = getTouchDist(e.touches);
                    var factor = dist / pinchStartDist;
                    var newScale = Math.min(maxScale, Math.max(minScale, pinchStartScale * factor));
                    var midX = (e.touches[0].clientX + e.touches[1].clientX) / 2 - viewport.getBoundingClientRect().left;
                    var midY = (e.touches[0].clientY + e.touches[1].clientY) / 2 - viewport.getBoundingClientRect().top;
                    var ratio = newScale / pinchStartScale;
                    tx = midX - ratio * (midX - pinchStartTx);
                    ty = midY - ratio * (midY - pinchStartTy);
                    scale = newScale;
                    applyTransform();
                }
                e.preventDefault();
            }
            function onTouchEnd(e) {
                if (e.touches.length === 0) { isPanning = false; }
            }

            window.openMediaModal = function(el) {
                if (!getModal()) return;

                var isSvg = el.tagName.toLowerCase() === 'svg';
                var intrinsicW = 0, intrinsicH = 0;
                if (isSvg) {
                    // Resolve intrinsic pixel dimensions from viewBox or bounding rect.
                    // Mermaid sets width="100%" on the SVG; viewBox always carries the
                    // true pixel dimensions, so prefer it over the attribute value.
                    var vb = el.viewBox && el.viewBox.baseVal;
                    if (vb && vb.width && vb.height) {
                        intrinsicW = vb.width;
                        intrinsicH = vb.height;
                    }
                    if (!intrinsicW || !intrinsicH) {
                        var br = el.getBoundingClientRect();
                        intrinsicW = br.width;
                        intrinsicH = br.height;
                    }
                } else {
                    // For <img> use naturalWidth/naturalHeight (decoded pixel size).
                    intrinsicW = el.naturalWidth  || 0;
                    intrinsicH = el.naturalHeight || 0;
                    if (!intrinsicW || !intrinsicH) {
                        var ibr = el.getBoundingClientRect();
                        intrinsicW = ibr.width;
                        intrinsicH = ibr.height;
                    }
                }

                // Move the original element into the modal canvas instead of cloning.
                // For SVGs, cloning duplicates all id attributes, which breaks SVG
                // fragment references (url(#…)) because they resolve document-wide.
                // For consistency, images also use the same move+placeholder strategy.
                _activeSvg = el;
                _activeSvgParent = el.parentNode;
                _activeSvgOrigWidth  = el.getAttribute('width');
                _activeSvgOrigHeight = el.getAttribute('height');
                _activeSvgOrigStyle  = el.getAttribute('style');
                el.removeAttribute('style');
                if (isSvg && intrinsicW && intrinsicH) {
                    // Stamp explicit px dimensions so fitToScreen has a stable size.
                    el.setAttribute('width',  intrinsicW);
                    el.setAttribute('height', intrinsicH);
                }
                canvas.innerHTML = '';
                // Insert a placeholder with the same dimensions so the page
                // layout does not collapse/reflow while the element is in the modal.
                // Use <span> (not <div>) so the placeholder is valid in all parent
                // contexts an image or SVG can appear in (e.g. <p>, <span>, <a>).
                var elRect = el.getBoundingClientRect();
                var placeholder = document.createElement('span');
                placeholder.className = 'media-placeholder';
                placeholder.style.display = 'inline-block';
                placeholder.style.width  = elRect.width  + 'px';
                placeholder.style.height = elRect.height + 'px';
                _activeSvgParent.insertBefore(placeholder, el);
                _activeSvgPlaceholder = placeholder;
                canvas.appendChild(el);

                scale = 1; tx = 0; ty = 0;
                applyTransform();
                modal.classList.add('is-open');
                _prevBodyOverflow = document.body.style.overflow;
                document.body.style.overflow = 'hidden';
                modalOpen = true;
                _previousFocus = document.activeElement;
                requestAnimationFrame(function() {
                    fitToScreen();
                    // Focus the first focusable element so the focus trap works
                    // correctly from the start (including Shift+Tab).
                    var initialFocusEl = modal.querySelector(
                        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
                    );
                    (initialFocusEl || viewport).focus();
                });
                _boundMouseMove = onMouseMove;
                _boundMouseUp   = onMouseUp;
                _boundWheel     = onWheel;
                _boundKeyDown   = onKeyDown;
                _boundTouchMove = onTouchMove;
                _boundTouchEnd  = onTouchEnd;
                viewport.addEventListener('mousedown',  onMouseDown);
                document.addEventListener('mousemove',  _boundMouseMove);
                document.addEventListener('mouseup',    _boundMouseUp);
                viewport.addEventListener('wheel',      _boundWheel, { passive: false });
                document.addEventListener('keydown',    _boundKeyDown);
                viewport.addEventListener('touchstart', onTouchStart, { passive: false });
                viewport.addEventListener('touchmove',  _boundTouchMove, { passive: false });
                viewport.addEventListener('touchend',   _boundTouchEnd);
            };

            window.closeMediaModal = function() {
                if (!modalOpen || !getModal()) return;
                modal.classList.remove('is-open');
                document.body.style.overflow = _prevBodyOverflow;
                modalOpen = false;
                isPanning = false;

                // Move the element back to its original parent and restore attributes.
                if (_activeSvg && _activeSvgParent) {
                    if (_activeSvgOrigWidth !== null) {
                        _activeSvg.setAttribute('width', _activeSvgOrigWidth);
                    } else {
                        _activeSvg.removeAttribute('width');
                    }
                    if (_activeSvgOrigHeight !== null) {
                        _activeSvg.setAttribute('height', _activeSvgOrigHeight);
                    } else {
                        _activeSvg.removeAttribute('height');
                    }
                    if (_activeSvgOrigStyle !== null) {
                        _activeSvg.setAttribute('style', _activeSvgOrigStyle);
                    } else {
                        _activeSvg.removeAttribute('style');
                    }
                    // Reinsert the element at the placeholder's position to preserve original ordering.
                    if (_activeSvgPlaceholder && _activeSvgPlaceholder.parentNode) {
                        _activeSvgPlaceholder.parentNode.insertBefore(_activeSvg, _activeSvgPlaceholder);
                        _activeSvgPlaceholder.parentNode.removeChild(_activeSvgPlaceholder);
                    } else {
                        // Fallback: if no placeholder is available, insert as first child.
                        _activeSvgParent.insertBefore(_activeSvg, _activeSvgParent.firstChild);
                    }
                    _activeSvgPlaceholder = null;
                }
                _activeSvg = null;
                _activeSvgParent = null;
                _activeSvgOrigWidth = null;
                _activeSvgOrigHeight = null;
                _activeSvgOrigStyle = null;
                canvas.innerHTML = '';

                viewport.removeEventListener('mousedown',  onMouseDown);
                document.removeEventListener('mousemove',  _boundMouseMove);
                document.removeEventListener('mouseup',    _boundMouseUp);
                viewport.removeEventListener('wheel',      _boundWheel);
                document.removeEventListener('keydown',    _boundKeyDown);
                viewport.removeEventListener('touchstart', onTouchStart);
                viewport.removeEventListener('touchmove',  _boundTouchMove);
                viewport.removeEventListener('touchend',   _boundTouchEnd);

                // Restore focus to the element that triggered the modal.
                if (_previousFocus && typeof _previousFocus.focus === 'function') {
                    _previousFocus.focus();
                }
                _previousFocus = null;
            };
        }());

        function initMermaidExpand() {
            var containers = document.querySelectorAll('.prose pre.mermaid');
            containers.forEach(function(pre) {
                if (pre.querySelector('.mermaid-expand-btn')) return;
                var svg = pre.querySelector(':scope > svg');
                if (!svg) return;
                var btn = document.createElement('button');
                btn.type = 'button';
                btn.className = 'mermaid-expand-btn';
                btn.setAttribute('aria-label', 'View diagram fullscreen');
                btn.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/></svg><span>Expand</span>';
                btn.addEventListener('click', function(e) {
                    e.stopPropagation();
                    var s = pre.querySelector(':scope > svg');
                    if (s) { window.openMediaModal(s); }
                });
                pre.appendChild(btn);
            });
        }

        function initImageExpand() {
            var images = document.querySelectorAll('.prose img');
            images.forEach(function(img) {
                // Determine what to wrap: if the direct parent is an <a>, wrap the
                // <a> so the link and expand button coexist inside the wrapper.
                var target = (img.parentNode && img.parentNode.tagName.toLowerCase() === 'a')
                    ? img.parentNode
                    : img;
                // Idempotency: skip if the target is already inside a wrapper.
                if (target.parentNode && target.parentNode.classList.contains('img-expand-wrapper')) return;
                // Build the wrapper <span>.
                var wrapper = document.createElement('span');
                wrapper.className = 'img-expand-wrapper';
                target.parentNode.insertBefore(wrapper, target);
                wrapper.appendChild(target);
                // Build the expand button.
                var btn = document.createElement('button');
                btn.type = 'button';
                btn.className = 'img-expand-btn';
                btn.setAttribute('aria-label', 'View image fullscreen');
                btn.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/></svg><span>Expand</span>';
                btn.addEventListener('click', function(e) {
                    e.stopPropagation();
                    window.openMediaModal(img);
                });
                wrapper.appendChild(btn);
            });
        }
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

    <!-- Media fullscreen viewer modal (mermaid diagrams + images) -->
    <div id="media-modal" role="dialog" aria-modal="true" aria-label="Media viewer">
        <div id="media-modal-header">
            <button id="media-modal-close" aria-label="Close media viewer">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
        </div>
        <div id="media-modal-viewport" tabindex="-1">
            <div id="media-modal-canvas"></div>
        </div>
        <div id="media-modal-controls">
            <button class="media-ctrl-btn" id="media-zoom-in" aria-label="Zoom in">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/><line x1="11" y1="8" x2="11" y2="14"/><line x1="8" y1="11" x2="14" y2="11"/></svg>
            </button>
            <span id="media-zoom-level" aria-live="polite" aria-label="Zoom level">100%</span>
            <button class="media-ctrl-btn" id="media-zoom-out" aria-label="Zoom out">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/><line x1="8" y1="11" x2="14" y2="11"/></svg>
            </button>
            <button class="media-ctrl-btn" id="media-zoom-reset" aria-label="Fit to screen" style="width: auto; padding: 0 0.5rem; font-size: 0.7rem; font-weight: 500; letter-spacing: 0.02em;">Fit</button>
        </div>
    </div>
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
            <h3 class="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">
                <a href="/docs/{{.Doc.Repo}}/"
                   hx-get="/docs/{{.Doc.Repo}}/" hx-target="#main-content" hx-push-url="true"
                   class="block hover:text-blue-600 transition-colors">{{.Doc.Repo}}</a>
            </h3>
            <ul class="space-y-1">
                {{template "sidebarDocTree" (sidebarNav .NavDocs .CurrentPath)}}
            </ul>
        </nav>
    </aside>
    <article id="doc-content" class="flex-1 min-w-0">
        <div class="mb-4 text-sm text-gray-500 flex items-center justify-between">
            <div>
                <a href="/" hx-get="/" hx-target="#main-content" hx-push-url="true" class="hover:text-blue-600">Home</a>
                <span class="mx-1">/</span>
                <a href="/docs/{{.Doc.Repo}}/" hx-get="/docs/{{.Doc.Repo}}/" hx-target="#main-content" hx-push-url="true" class="hover:text-blue-600">{{.Doc.Repo}}</a>
                <span class="mx-1">/</span>
                <span>{{.Doc.Path}}</span>
            </div>
            <a href="{{githubURL .Doc.Repo .Doc.Path .Doc.CommitSHA}}" target="_blank" rel="noopener noreferrer"
               class="inline-flex items-center gap-1 text-gray-400 hover:text-blue-600 transition-colors">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                View source
            </a>
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
    <style>
    .search-result mark { background-color: #dbeafe; color: #1e3a8a; border-radius: 2px; padding: 0 2px; }
    </style>
    <div class="space-y-4">
        {{range .Results.Hits}}
        <a href="/docs/{{.Repo}}/{{.Path}}{{if .Anchor}}#{{.Anchor}}{{end}}" hx-get="/docs/{{.Repo}}/{{.Path}}" hx-target="#main-content" hx-push-url="/docs/{{.Repo}}/{{.Path}}{{if .Anchor}}#{{.Anchor}}{{end}}"
           class="search-result block p-4 bg-white rounded-lg border border-gray-200 hover:border-blue-500 hover:shadow-sm transition-all">
            <h3 class="text-lg font-semibold text-gray-900 mb-1">
                {{- if .TitleFragments -}}
                    {{- range $i, $f := .TitleFragments -}}
                        {{- if $i}}<span class="text-gray-300 mx-1">&hellip;</span>{{end -}}
                        {{safeFragment $f}}
                    {{- end -}}
                {{- else -}}
                    {{.Title}}
                {{- end -}}
            </h3>
            <p class="text-xs text-gray-400 mb-2">{{.Repo}}/{{.Path}}</p>
            {{if .ContentFragments}}
            <p class="text-sm text-gray-600 leading-relaxed">
                {{- range $i, $f := .ContentFragments -}}
                    {{- if $i}}<span class="text-gray-300 mx-1">&hellip;</span>{{end -}}
                    {{safeFragment $f}}
                {{- end -}}
            </p>
            {{else if .TitleFragments}}
            <p class="text-xs text-gray-400 italic">Matched in title</p>
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
    <div class="space-y-1">
        {{template "repoDocTree" .Docs}}
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
            <h3 class="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">
                <a href="/docs/{{.Doc.Repo}}/"
                   hx-get="/docs/{{.Doc.Repo}}/" hx-target="#main-content" hx-push-url="true"
                   class="block hover:text-blue-600 transition-colors">{{.Doc.Repo}}</a>
            </h3>
            <ul class="space-y-1">
                {{template "sidebarDocTree" (sidebarNav .NavDocs .CurrentPath)}}
            </ul>
        </nav>
    </aside>
    <article id="doc-content" class="flex-1 min-w-0">
        <div class="mb-4 text-sm text-gray-500 flex items-center justify-between">
            <div>
                <a href="/" hx-get="/" hx-target="#main-content" hx-push-url="true" class="hover:text-blue-600">Home</a>
                <span class="mx-1">/</span>
                <a href="/docs/{{.Doc.Repo}}/" hx-get="/docs/{{.Doc.Repo}}/" hx-target="#main-content" hx-push-url="true" class="hover:text-blue-600">{{.Doc.Repo}}</a>
                <span class="mx-1">/</span>
                <span>{{.Doc.Path}}</span>
            </div>
            <a href="{{githubURL .Doc.Repo .Doc.Path .Doc.CommitSHA}}" target="_blank" rel="noopener noreferrer"
               class="inline-flex items-center gap-1 text-gray-400 hover:text-blue-600 transition-colors">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                View source
            </a>
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
                    if (typeof window.Scalar === 'undefined' || typeof window.Scalar.createApiReference !== 'function') return;
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

// repoDocTreeSubTemplate is a recursive named sub-template that renders a []DocNode
// as a directory tree for the repo index page.
// Folder nodes render as a heading followed by an indented subtree.
// Document nodes render as a clickable card row with title and updated date.
const repoDocTreeSubTemplate = `{{define "repoDocTree"}}
{{range .}}
{{if .Doc}}
<a href="/docs/{{.Doc.Repo}}/{{.Doc.Path}}"
   hx-get="/docs/{{.Doc.Repo}}/{{.Doc.Path}}" hx-target="#main-content" hx-push-url="true"
   class="flex items-center justify-between p-4 bg-white rounded-lg border border-gray-200 hover:border-blue-500 hover:shadow-sm transition-all mb-2">
    <h2 class="text-lg font-semibold text-gray-900">{{.Doc.Title}}</h2>
    <span class="text-sm text-gray-500 shrink-0 ml-4">Updated {{.Doc.UpdatedAt.Format "Jan 02, 2006"}}</span>
</a>
{{else}}
<div class="mt-4 mb-1">
    <div class="flex items-center gap-1.5 px-1 py-1 text-sm font-medium text-gray-500">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
        {{.Name}}
    </div>
    <div class="pl-4 border-l border-gray-200 ml-2">
        {{template "repoDocTree" .Children}}
    </div>
</div>
{{end}}
{{end}}
{{end}}`

// sidebarDocTreeSubTemplate is a recursive named sub-template that renders a []DocNode
// as a directory tree for the sidebar navigation on the document reading page.
// Folder nodes render as a non-clickable label followed by an indented subtree.
// Document nodes render as clickable links.
const sidebarDocTreeSubTemplate = `{{define "sidebarDocTree"}}
{{range .Nodes}}
{{if .Doc}}
<li>
    <a href="/docs/{{.Doc.Repo}}/{{.Doc.Path}}"
       hx-get="/docs/{{.Doc.Repo}}/{{.Doc.Path}}" hx-target="#main-content" hx-push-url="true"
       class="block px-3 py-1.5 text-sm rounded-md {{if eq .Doc.Path $.CurrentPath}}bg-blue-50 text-blue-700 font-medium{{else}}text-gray-700 hover:bg-gray-100 hover:text-gray-900{{end}}">
        {{.Doc.Title}}
    </a>
</li>
{{else}}
<li class="mt-2">
    <div class="flex items-center gap-1 px-3 py-1 text-sm font-medium text-gray-500">
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
        {{.Name}}
    </div>
    <ul class="pl-1 border-l border-gray-200 ml-1 space-y-1">
        {{template "sidebarDocTree" (sidebarChildren .Children $.CurrentPath)}}
    </ul>
</li>
{{end}}
{{end}}
{{end}}`
