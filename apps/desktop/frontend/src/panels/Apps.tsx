import { AppWindow } from 'lucide-react'
import { useBackend } from '../lib/backend'
import { Card, CardHeader } from '../components/ui'

const APPS = [
  { name: 'Notepad', path: 'notepad.exe' },
  { name: 'Calculator', path: 'calc.exe' },
  { name: 'Task Manager', path: 'taskmgr.exe' },
  { name: 'Command Prompt', path: 'cmd.exe' },
  { name: 'Paint', path: 'mspaint.exe' },
  { name: 'Explorer', path: 'explorer.exe' },
]

export function Apps() {
  const { send } = useBackend()
  return (
    <Card className="p-6">
      <CardHeader title="Launch apps" subtitle="Start a predefined application on the PC" />
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        {APPS.map((app) => (
          <button
            key={app.name}
            onClick={() => send('apps', 'launch', { name: app.name })}
            className="flex flex-col items-center gap-3 rounded-lg border border-stroke bg-control p-4 text-center transition-colors duration-150 hover:bg-control-hover cursor-pointer"
          >
            <span className="grid h-11 w-11 place-items-center rounded-lg bg-card text-accent"><AppWindow size={20} /></span>
            <div>
              <div className="text-sm font-medium text-text">{app.name}</div>
              <div className="font-mono text-[10px] text-text-tertiary">{app.path}</div>
            </div>
          </button>
        ))}
      </div>
    </Card>
  )
}
