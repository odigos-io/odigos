import { useQuery } from '@apollo/client';
import { useEffect, useState } from 'react';
import { GET_DESTINATION_TYPE } from '@/graphql';
import { DestinationsCategory, GetDestinationTypesResponse } from '@/types';

const CATEGORIES_DESCRIPTION = {
  managed: 'Effortless Monitoring with Scalable Performance Management',
  'self hosted': 'Full Control and Customization for Advanced Application Monitoring',
};

export interface IDestinationListItem extends DestinationsCategory {
  description: string;
}

const data = {
  destinationTypes: {
    categories: [
      {
        name: 'managed',
        items: [
          {
            type: 'appdynamics',
            testConnectionSupported: false,
            displayName: 'AppDynamics',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/appdynamics.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 's3',
            testConnectionSupported: false,
            displayName: 'AWS S3',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/s3.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'axiom',
            testConnectionSupported: false,
            displayName: 'Axiom',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/axiom.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'azureblob',
            testConnectionSupported: false,
            displayName: 'Azure Blob Storage',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/blobstorage.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'betterstack',
            testConnectionSupported: false,
            displayName: 'Better Stack',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/betterstack.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: false,
              },
            },
          },
          {
            type: 'causely',
            testConnectionSupported: false,
            displayName: 'Causely',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/causely.svg',
            supportedSignals: {
              logs: {
                supported: false,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'chronosphere',
            testConnectionSupported: false,
            displayName: 'Chronosphere',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/chronosphere.svg',
            supportedSignals: {
              logs: {
                supported: false,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'coralogix',
            testConnectionSupported: false,
            displayName: 'Coralogix',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/coralogix.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'dash0',
            testConnectionSupported: false,
            displayName: 'Dash0',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/dash0.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'datadog',
            testConnectionSupported: false,
            displayName: 'Datadog',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/datadog.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'dynatrace',
            testConnectionSupported: true,
            displayName: 'Dynatrace',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/dynatrace.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'elasticapm',
            testConnectionSupported: true,
            displayName: 'Elastic APM',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/elasticapm.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'qryn',
            testConnectionSupported: false,
            displayName: 'Gigapipe',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/gigapipe.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'googlecloud',
            testConnectionSupported: false,
            displayName: 'Google Cloud Monitoring',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/gcp.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'grafanacloudloki',
            testConnectionSupported: false,
            displayName: 'Grafana Cloud Loki',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/grafana.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: false,
              },
            },
          },
          {
            type: 'grafanacloudprometheus',
            testConnectionSupported: false,
            displayName: 'Grafana Cloud Prometheus',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/grafana.svg',
            supportedSignals: {
              logs: {
                supported: false,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: false,
              },
            },
          },
          {
            type: 'grafanacloudtempo',
            testConnectionSupported: false,
            displayName: 'Grafana Cloud Tempo',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/grafana.svg',
            supportedSignals: {
              logs: {
                supported: false,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'groundcover',
            testConnectionSupported: false,
            displayName: 'Groundcover inCloud',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/groundcover.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'honeycomb',
            testConnectionSupported: false,
            displayName: 'Honeycomb',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/honeycomb.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'hyperdx',
            testConnectionSupported: false,
            displayName: 'HyperDX',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/hyperdx.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'kloudmate',
            testConnectionSupported: false,
            displayName: 'KloudMate',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/kloudmate.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'last9',
            testConnectionSupported: false,
            displayName: 'Last9',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/last9.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'lightstep',
            testConnectionSupported: false,
            displayName: 'Lightstep',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/lightstep.svg',
            supportedSignals: {
              logs: {
                supported: false,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'logzio',
            testConnectionSupported: false,
            displayName: 'Logz.io',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/logzio.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'lumigo',
            testConnectionSupported: false,
            displayName: 'Lumigo',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/lumigo.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'middleware',
            testConnectionSupported: false,
            displayName: 'Middleware',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/middleware.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'newrelic',
            testConnectionSupported: true,
            displayName: 'New Relic',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/newrelic.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'opsverse',
            testConnectionSupported: false,
            displayName: 'OpsVerse',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/opsverse.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'splunk',
            testConnectionSupported: false,
            displayName: 'Splunk',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/splunk.svg',
            supportedSignals: {
              logs: {
                supported: false,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'sumologic',
            testConnectionSupported: false,
            displayName: 'Sumo Logic',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/sumologic.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'traceloop',
            testConnectionSupported: false,
            displayName: 'Traceloop',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/traceloop.svg',
            supportedSignals: {
              logs: {
                supported: false,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'uptrace',
            testConnectionSupported: false,
            displayName: 'Uptrace',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/uptrace.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
        ],
      },
      {
        name: 'self hosted',
        items: [
          {
            type: 'clickhouse',
            testConnectionSupported: false,
            displayName: 'Clickhouse',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/clickhouse.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'elasticsearch',
            testConnectionSupported: false,
            displayName: 'Elasticsearch',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/elasticsearch.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'jaeger',
            testConnectionSupported: true,
            displayName: 'Jaeger',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/jaeger.svg',
            supportedSignals: {
              logs: {
                supported: false,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'loki',
            testConnectionSupported: false,
            displayName: 'Loki',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/loki.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: false,
              },
            },
          },
          {
            type: 'otlp',
            testConnectionSupported: false,
            displayName: 'OTLP gRPC',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/opentelemetry.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'otlphttp',
            testConnectionSupported: false,
            displayName: 'OTLP http',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/opentelemetry.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'prometheus',
            testConnectionSupported: false,
            displayName: 'Prometheus',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/prometheus.svg',
            supportedSignals: {
              logs: {
                supported: false,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: false,
              },
            },
          },
          {
            type: 'qryn-oss',
            testConnectionSupported: false,
            displayName: 'qryn',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/qryn.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'quickwit',
            testConnectionSupported: false,
            displayName: 'Quickwit',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/quickwit.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'signoz',
            testConnectionSupported: false,
            displayName: 'SigNoz',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/signoz.svg',
            supportedSignals: {
              logs: {
                supported: true,
              },
              metrics: {
                supported: true,
              },
              traces: {
                supported: true,
              },
            },
          },
          {
            type: 'tempo',
            testConnectionSupported: false,
            displayName: 'Tempo',
            imageUrl: 'https:/d15jtxgb40qetw.cloudfront.net/tempo.svg',
            supportedSignals: {
              logs: {
                supported: false,
              },
              metrics: {
                supported: false,
              },
              traces: {
                supported: true,
              },
            },
          },
        ],
      },
    ],
  },
};

export function useDestinationTypes() {
  const [destinations, setDestinations] = useState<IDestinationListItem[]>([]);
  // const { data } = useQuery<GetDestinationTypesResponse>(GET_DESTINATION_TYPE);

  useEffect(() => {
    if (data) {
      setDestinations(
        // @ts-ignore
        data.destinationTypes.categories.map((category) => ({
          name: category.name,
          description: CATEGORIES_DESCRIPTION[category.name as keyof typeof CATEGORIES_DESCRIPTION],
          items: category.items,
        })),
      );
    }
  }, [data]);

  return { destinations };
}
