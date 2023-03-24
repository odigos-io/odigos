import {ObservabilitySignals, ObservabilityVendor, VendorType} from "@/vendors/index";
import QrynLogo from "@/img/vendor/qryn.svg";
import {NextApiRequest} from "next";

export class Qryn implements ObservabilityVendor {
    name = "qryn";
    displayName = "qryn";
    type = VendorType.MANAGED;
    supportedSignals = [
        ObservabilitySignals.Traces,
        ObservabilitySignals.Metrics,
        ObservabilitySignals.Logs,
    ];

    getLogo = (props: any) => {
        return <QrynLogo {...props} />;
    };

    getFields = (selectedSignals: any) => {
        return [
            {
                displayName: "qryn API URL",
                id: "url",
                name: "url",
                type: "url",
            },
            {
                displayName: "qryn API Key",
                id: "apiKey",
                name: "apiKey",
                type: "text",
            },
            {
                displayName: "qryn API Secret",
                id: "apiSecret",
                name: "apiSecret",
                type: "password",
            },
        ];
    };

    toObjects = (req: NextApiRequest) => {
        return {
            Data: {
                QRYN_URL: req.body.url,
                QRYN_API_KEY: req.body.apiKey
            },
            Secret: {
                QRYN_API_SECRET: Buffer.from(req.body.apiSecret).toString("base64")
            },
        };
    };

    mapDataToFields = (data: any) => {
        return {
            url: data.QRYN_URL || "qryn.gigapipe.com",
        };
    };
}
