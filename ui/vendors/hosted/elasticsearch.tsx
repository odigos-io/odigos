import {
    ObservabilityVendor,
    ObservabilitySignals,
    VendorObjects,
    VendorType,
} from "@/vendors/index";
import ElasticsearchLogo from "@/img/vendor/elasticsearch.svg";
import { NextApiRequest } from "next";

export class Elasticsearch implements ObservabilityVendor {
    name = "elasticsearch";
    displayName = "Elasticsearch";
    type = VendorType.HOSTED;
    supportedSignals = [ObservabilitySignals.Traces, ObservabilitySignals.Logs];

    getLogo = (props: any) => {
        return <ElasticsearchLogo {...props} />;
    };

    getFields = (selectedSignals: any) => {
        let fields = [
            {
                displayName: "Elasticsearch Endpoint",
                id: "elasticsearch_url",
                name: "elasticsearch_url",
                type: "url",
            },
        ];

        if (selectedSignals[ObservabilitySignals.Traces]) {
            fields.push(
                {
                    displayName: "Traces Index",
                    id: "traces_index",
                    name: "traces_index",
                    type: "text",
                }
            );
        }

        if (selectedSignals[ObservabilitySignals.Logs]) {
            fields.push(
                {
                    displayName: "Logs Index",
                    id: "logs_index",
                    name: "logs_index",
                    type: "text",
                }
            );
        }

        return fields;
    };

    toObjects = (req: NextApiRequest) => {
        return {
            Data: {
                ELASTICSEARCH_URL: req.body.elasticsearch_url,
                ES_TRACES_INDEX: req.body.traces_index,
                ES_LOGS_INDEX: req.body.logs_index,
            },
        };
    };

    mapDataToFields = (data: any) => {
        return {
            elasticsearch_url: data.ELASTICSEARCH_URL,
            traces_index: data.ES_TRACES_INDEX || "",
            logs_index: data.ES_LOGS_INDEX || "",
        };
    };
}
