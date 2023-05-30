import { ObservabilitySignals, ObservabilityVendor, VendorType } from '@/vendors/index'
import VictoriaMetricsLogo from '@/img/vendor/victoriametrics.svg'
import { NextApiRequest } from 'next'

export class VictoriaMetrics implements ObservabilityVendor {
  name = 'victoriametrics'
  displayName = 'Victoria Metrics'
  type = VendorType.HOSTED
  supportedSignals = [ObservabilitySignals.Traces, ObservabilitySignals.Metrics]

  getLogo = (props: any) => {
    return <VictoriaMetricsLogo {...props} />
  }

  getFields = () => {
    return [
      {
        displayName: 'URL',
        id: 'url',
        name: 'url',
        type: 'url',
      },
      {
        displayName: 'PORT',
        id: 'port',
        name: 'port',
        type: 'text',
      },
      {
        displayName: 'API',
        id: 'api',
        name: 'api',
        type: 'text',
      },
    ]
  }

  toObjects = (req: NextApiRequest) => {
    return {
      Data: {
        VICTORIA_METRICS_URL: req.body.url,
        VICTORIA_METRICS_PORT: req.body.port,
        VICTORIA_METRICS_PROM_API: req.body.api,
      },
    }
  }

  mapDataToFields = (data: any) => {
    return {
      url: data.VICTORIA_METRICS_URL,
      port: data.VICTORIA_METRICS_PORT,
      api: data.VICTORIA_METRICS_PROM_API,
    }
  }
}
