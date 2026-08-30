import { useEffect } from 'react'

// No SSR/per-route meta in this SPA, so at minimum give each marketing page
// its own browser-tab title (bookmarks, tab-switching, back/forward all
// benefit) rather than every route showing the same static <title>.
export function useDocumentTitle(title) {
  useEffect(() => {
    const prev = document.title
    document.title = title
    return () => { document.title = prev }
  }, [title])
}
