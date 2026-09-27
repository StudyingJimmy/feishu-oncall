import { useEffect, useState } from 'react'
import MobileWorkspace from '../mobile/MobileWorkspace'
import Workspace from './Workspace'
import { useOncallWorkspace } from './useOncallWorkspace'

function shouldUseMobile(): boolean {
  const requestedView = new URLSearchParams(window.location.search).get('view')
  if (requestedView === 'desktop') return false
  if (requestedView === 'mobile') return true
  return window.location.pathname.replace(/\/$/, '').endsWith('/mobile') || window.matchMedia('(max-width: 760px)').matches
}

export default function ResponsiveWorkspace() {
  const workspace = useOncallWorkspace()
  const [mobile, setMobile] = useState(shouldUseMobile)

  useEffect(() => {
    const media = window.matchMedia('(max-width: 760px)')
    const update = () => setMobile(shouldUseMobile())
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [])

  return mobile ? <MobileWorkspace workspace={workspace} /> : <Workspace workspace={workspace} />
}
