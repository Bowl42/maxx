import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui'
import { useTranslation } from 'react-i18next'
import {
  useProviders,
  useRoutes,
  useProjects,
  useProxyRequests,
} from '@/hooks/queries'
import {
  Activity,
  Server,
  Route,
  FolderKanban,
  Zap,
  ArrowRight,
  CheckCircle,
  XCircle,
  Ban,
  LayoutDashboard,
} from 'lucide-react'
import { Link } from 'react-router-dom'
import { PageHeader } from '@/components/layout/page-header'

export function OverviewPage() {
  const { t } = useTranslation()
  const { data: providers } = useProviders()
  const { data: routes } = useRoutes()
  const { data: projects } = useProjects()
  const { data: requestsData } = useProxyRequests({ limit: 10 })

  const requests = requestsData?.items ?? []

  const stats = [
    {
      label: t('dashboard.providers'),
      value: providers?.length ?? 0,
      icon: Server,
      className: 'text-blue-600 dark:text-blue-400 bg-muted',
      href: '/providers',
    },
    {
      label: t('dashboard.routes'),
      value: routes?.length ?? 0,
      icon: Route,
      className: 'text-violet-600 dark:text-violet-400 bg-muted',
      href: '/routes/claude',
    },
    {
      label: t('dashboard.projects'),
      value: projects?.length ?? 0,
      icon: FolderKanban,
      className: 'text-amber-600 dark:text-amber-400 bg-muted',
      href: '/projects',
    },
    {
      label: t('dashboard.recentRequests'),
      value: requests.length,
      icon: Activity,
      className: 'text-emerald-600 dark:text-emerald-400 bg-muted',
      href: '/requests',
    },
  ]

  const completedRequests = requests.filter(
    r => r.status === 'COMPLETED'
  ).length
  const failedRequests = requests.filter(r => r.status === 'FAILED').length
  const cancelledRequests = requests.filter(
    r => r.status === 'CANCELLED'
  ).length
  const hasProviders = (providers?.length ?? 0) > 0

  return (
    <div className="flex flex-col h-full">
      <PageHeader
        icon={LayoutDashboard}
        iconClassName="text-indigo-500"
        title={t('dashboard.title')}
        description={t('dashboard.description')}
      />
      <div className="flex-1 overflow-y-auto p-4 md:p-6">
        <div className="space-y-6 md:space-y-8 animate-fade-in max-w-7xl mx-auto">
          {/* Welcome Section */}
          {!hasProviders && (
            <div className="text-center py-16 md:py-20 px-4">
              <div className="w-24 h-24 rounded-3xl bg-linear-to-br from-violet-500 to-indigo-600 flex items-center justify-center mx-auto mb-8 shadow-xl shadow-indigo-500/20 animate-pulse-slow ring-4 ring-white/50 dark:ring-white/10">
                <Zap size={40} className="text-white drop-shadow-md" />
              </div>
              <h1 className="text-3xl md:text-5xl font-bold text-foreground mb-6 tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-violet-600 to-indigo-600 dark:from-violet-400 dark:to-indigo-400">
                {t('dashboard.welcome')}
              </h1>
              <p className="text-base md:text-lg text-muted-foreground max-w mx-auto mb-10 leading-relaxed">
                {t('dashboard.welcomeDescription')}
              </p>
              <Link
                to="/providers"
                className="inline-flex items-center gap-2 bg-gradient-to-r from-violet-600 to-indigo-600 text-white px-8 py-3 rounded-xl hover:opacity-90 transition-all duration-300 font-medium text-sm shadow-lg shadow-indigo-500/25 hover:shadow-xl hover:shadow-indigo-500/30 hover:scale-105 active:scale-95"
              >
                {t('dashboard.getStarted')}
                <ArrowRight className="h-4 w-4" />
              </Link>
            </div>
          )}

          {/* Stats Grid */}
          <div className="grid gap-4 md:gap-6 grid-cols-2 lg:grid-cols-4">
            {stats.map(stat => {
              const Icon = stat.icon
              return (
                <Link key={stat.label} to={stat.href} className="group">
                  <Card className="h-full hover:shadow-lg hover:shadow-accent/5 cursor-pointer border-border/50 bg-card/50 backdrop-blur-sm transition-all duration-300 hover:border-accent/40 hover:-translate-y-1">
                    <CardContent className="p-4 md:p-6">
                      <div className="flex items-center justify-between gap-4">
                        <div className="flex-1 min-w-0 space-y-1">
                          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                            {stat.label}
                          </p>
                          <p className="text-2xl md:text-3xl font-bold text-foreground font-mono tracking-tight">
                            {stat.value}
                          </p>
                        </div>
                        <div
                          className={`p-3 rounded-2xl ${stat.className} transition-transform duration-300 group-hover:scale-110 group-hover:rotate-3 shadow-sm`}
                        >
                          <Icon className="h-5 w-5 md:h-6 md:w-6" />
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                </Link>
              )
            })}
          </div>

          {/* Status Cards */}
          <div className="grid gap-4 md:gap-6 grid-cols-1 md:grid-cols-2">
            <Card className="border-border/50 bg-card/50 backdrop-blur-sm transition-all duration-300 hover:shadow-lg hover:shadow-accent/5">
              <CardHeader className="border-b border-border/50 py-4 px-5">
                <CardTitle className="text-base font-semibold flex items-center gap-2">
                  <Activity className="h-4 w-4 text-emerald-500" />
                  {t('dashboard.requestStatus')}
                </CardTitle>
              </CardHeader>
              <CardContent className="p-5">
                <div className="space-y-3">
                  <div className="flex items-center justify-between p-4 rounded-xl bg-muted/50 border border-border hover:bg-accent transition-colors group">
                    <div className="flex items-center gap-3">
                      <div className="p-2 rounded-lg bg-muted group-hover:bg-muted/80 transition-colors">
                        <CheckCircle className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
                      </div>
                      <span className="text-sm font-medium text-foreground">
                        {t('dashboard.completed')}
                      </span>
                    </div>
                    <span className="text-xl font-bold text-emerald-600 dark:text-emerald-400 font-mono tabular-nums">
                      {completedRequests}
                    </span>
                  </div>
                  <div className="flex items-center justify-between p-4 rounded-xl bg-muted/50 border border-border hover:bg-accent transition-colors group">
                    <div className="flex items-center gap-3">
                      <div className="p-2 rounded-lg bg-muted group-hover:bg-muted/80 transition-colors">
                        <XCircle className="h-4 w-4 text-red-600 dark:text-red-400" />
                      </div>
                      <span className="text-sm font-medium text-foreground">
                        {t('dashboard.failed')}
                      </span>
                    </div>
                    <span className="text-xl font-bold text-red-600 dark:text-red-400 font-mono tabular-nums">
                      {failedRequests}
                    </span>
                  </div>
                  <div className="flex items-center justify-between p-4 rounded-xl bg-muted/50 border border-border hover:bg-accent transition-colors group">
                    <div className="flex items-center gap-3">
                      <div className="p-2 rounded-lg bg-muted group-hover:bg-muted/80 transition-colors">
                        <Ban className="h-4 w-4 text-amber-600 dark:text-amber-400" />
                      </div>
                      <span className="text-sm font-medium text-foreground">
                        {t('dashboard.cancelled')}
                      </span>
                    </div>
                    <span className="text-xl font-bold text-amber-600 dark:text-amber-400 font-mono tabular-nums">
                      {cancelledRequests}
                    </span>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-border/50 bg-card/50 backdrop-blur-sm transition-all duration-300 hover:shadow-lg hover:shadow-accent/5">
              <CardHeader className="border-b border-border/50 py-4 px-5">
                <CardTitle className="text-base font-semibold flex items-center gap-2">
                  <Zap className="h-4 w-4 text-amber-500" />
                  {t('dashboard.quickActions')}
                </CardTitle>
              </CardHeader>
              <CardContent className="p-5">
                <div className="space-y-2">
                  <Link
                    to="/providers"
                    className="flex items-center justify-between p-4 rounded-xl bg-muted/50 border border-border hover:bg-accent hover:shadow-sm transition-all group"
                  >
                    <div className="flex items-center gap-3">
                      <div className="p-2 rounded-lg bg-muted text-blue-600 dark:text-blue-400 group-hover:bg-muted/80 transition-colors">
                        <Server className="h-4 w-4" />
                      </div>
                      <span className="text-sm font-medium text-foreground">
                        {t('dashboard.manageProviders')}
                      </span>
                    </div>
                    <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-foreground group-hover:translate-x-1 transition-all" />
                  </Link>
                  <Link
                    to="/routes"
                    className="flex items-center justify-between p-4 rounded-xl bg-muted/50 border border-border hover:bg-accent hover:shadow-sm transition-all group"
                  >
                    <div className="flex items-center gap-3">
                      <div className="p-2 rounded-lg bg-muted text-violet-600 dark:text-violet-400 group-hover:bg-muted/80 transition-colors">
                        <Route className="h-4 w-4" />
                      </div>
                      <span className="text-sm font-medium text-foreground">
                        {t('dashboard.configureRoutes')}
                      </span>
                    </div>
                    <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-foreground group-hover:translate-x-1 transition-all" />
                  </Link>
                  <Link
                    to="/requests"
                    className="flex items-center justify-between p-4 rounded-xl bg-muted/50 border border-border hover:bg-accent hover:shadow-sm transition-all group"
                  >
                    <div className="flex items-center gap-3">
                      <div className="p-2 rounded-lg bg-muted text-emerald-600 dark:text-emerald-400 group-hover:bg-muted/80 transition-colors">
                        <Activity className="h-4 w-4" />
                      </div>
                      <span className="text-sm font-medium text-foreground">
                        {t('dashboard.viewRequests')}
                      </span>
                    </div>
                    <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-foreground group-hover:translate-x-1 transition-all" />
                  </Link>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Features */}
          {!hasProviders && (
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 md:gap-6">
              <div className="bg-card/50 backdrop-blur-sm border border-border/50 rounded-2xl p-6 md:p-8 text-center hover:bg-accent hover:border-border transition-all duration-300 group hover:-translate-y-1">
                <div className="w-12 h-12 md:w-14 md:h-14 rounded-xl bg-muted flex items-center justify-center mx-auto mb-4 group-hover:bg-muted/80 transition-colors">
                  <CheckCircle className="h-6 w-6 md:h-7 md:w-7 text-emerald-600 dark:text-emerald-400" />
                </div>
                <h3 className="text-base font-semibold text-foreground mb-2">
                  {t('dashboard.secure')}
                </h3>
                <p className="text-sm text-muted-foreground">
                  {t('dashboard.secureDesc')}
                </p>
              </div>
              <div className="bg-card/50 backdrop-blur-sm border border-border/50 rounded-2xl p-6 md:p-8 text-center hover:bg-accent hover:border-border transition-all duration-300 group hover:-translate-y-1">
                <div className="w-12 h-12 md:w-14 md:h-14 rounded-xl bg-muted flex items-center justify-center mx-auto mb-4 group-hover:bg-muted/80 transition-colors">
                  <Zap className="h-6 w-6 md:h-7 md:w-7 text-violet-600 dark:text-violet-400" />
                </div>
                <h3 className="text-base font-semibold text-foreground mb-2">
                  {t('dashboard.fast')}
                </h3>
                <p className="text-sm text-muted-foreground">
                  {t('dashboard.fastDesc')}
                </p>
              </div>
              <div className="bg-card/50 backdrop-blur-sm border border-border/50 rounded-2xl p-6 md:p-8 text-center hover:bg-accent hover:border-border transition-all duration-300 group hover:-translate-y-1">
                <div className="w-12 h-12 md:w-14 md:h-14 rounded-xl bg-muted flex items-center justify-center mx-auto mb-4 group-hover:bg-muted/80 transition-colors">
                  <Activity className="h-6 w-6 md:h-7 md:w-7 text-blue-600 dark:text-blue-400" />
                </div>
                <h3 className="text-base font-semibold text-foreground mb-2">
                  {t('dashboard.insights')}
                </h3>
                <p className="text-sm text-muted-foreground">
                  {t('dashboard.insightsDesc')}
                </p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
