import {
    ObservabilityVendor,
    ObservabilitySignals,
    VendorType,
} from "@/vendors/index";
import GCSLogo from "@/img/vendor/gcs.svg";
import { NextApiRequest } from "next";

export class GoogleCloudStorage implements ObservabilityVendor {
    name = "gcs";
    displayName = "Google Cloud Storage";
    type = VendorType.MANAGED;
    supportedSignals = [ObservabilitySignals.Traces, ObservabilitySignals.Logs];
    getLogo = (props: any) => {
        return <GCSLogo {...props} />;
    };

    getFields = (selectedSignals: any) => {
        return [
            {
                displayName: "Bucket Name",
                id: "bucket",
                name: "bucket",
                type: "text",
            }
        ];
    };

    toObjects = (req: NextApiRequest) => {
        return {
            Data: {
                GCS_BUCKET: req.body.bucket,
            }
        };
    };

    mapDataToFields = (data: any) => {
        return {
            bucket: data.GCS_BUCKET || "",
        };
    };
}
