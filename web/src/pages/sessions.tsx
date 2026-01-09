import { Badge, Card, CardContent, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui';
import { useSessions } from '@/hooks/queries';
import { LayoutDashboard, Loader2, Calendar } from 'lucide-react';

export function SessionsPage() {
  const { data: sessions, isLoading } = useSessions();

  return (
    <div className="flex flex-col h-full bg-background">
       {/* Header */}
      <div className="h-[73px] flex items-center justify-between px-6 border-b border-border bg-surface-primary flex-shrink-0">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-accent/10 rounded-lg">
            <LayoutDashboard size={20} className="text-accent" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-text-primary leading-tight">Sessions</h2>
            <p className="text-xs text-text-secondary">
              Active client sessions
            </p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-auto p-6">
        <Card className="border-border bg-surface-primary">
          <CardContent className="p-0">
            {isLoading ? (
               <div className="flex items-center justify-center p-12">
                <Loader2 className="h-8 w-8 animate-spin text-accent" />
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent border-border">
                    <TableHead className="w-[100px] text-text-secondary">ID</TableHead>
                    <TableHead className="text-text-secondary">Session ID</TableHead>
                    <TableHead className="text-text-secondary">Client</TableHead>
                    <TableHead className="text-text-secondary">Project</TableHead>
                    <TableHead className="text-right text-text-secondary">Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sessions?.map((session) => (
                    <TableRow key={session.id} className="border-border hover:bg-surface-hover">
                      <TableCell className="font-mono text-xs text-text-muted">{session.id}</TableCell>
                      <TableCell className="font-mono text-xs text-text-primary">
                        {session.sessionID}
                      </TableCell>
                      <TableCell>
                        <Badge variant="info" className="capitalize">{session.clientType}</Badge>
                      </TableCell>
                      <TableCell>
                        {session.projectID === 0 ? (
                          <span className="text-text-muted text-xs italic">Default</span>
                        ) : (
                          <span className="font-mono text-xs text-text-primary">#{session.projectID}</span>
                        )}
                      </TableCell>
                      <TableCell className="text-right text-xs text-text-secondary font-mono">
                        {new Date(session.createdAt).toLocaleString()}
                      </TableCell>
                    </TableRow>
                  ))}
                  {(!sessions || sessions.length === 0) && (
                    <TableRow>
                      <TableCell colSpan={5} className="h-32 text-center text-text-muted">
                        <div className="flex flex-col items-center justify-center gap-2">
                           <Calendar className="h-8 w-8 opacity-20" />
                           <p>No active sessions</p>
                        </div>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
