import {
    ObservabilityVendor,
    ObservabilitySignals,
    VendorType,
} from "@/vendors/index";
import S3Logo from "@/img/vendor/s3.svg";
import { NextApiRequest } from "next";

export class AWSS3 implements ObservabilityVendor {
    name = "s3";
    displayName = "AWS S3";
    type = VendorType.MANAGED;
    supportedSignals = [ObservabilitySignals.Traces, ObservabilitySignals.Logs];
    getLogo = (props: any) => {
        return <S3Logo {...props} />;
    };

    getFields = (selectedSignals: any) => {
        return [
            {
                displayName: "Bucket Name",
                id: "bucket",
                name: "bucket",
                type: "text",
            }, 
            {
                displayName: "Bucket Region",
                id: "region",
                name: "region",
                type: "text",
            }
        ];
    };

    toObjects = (req: NextApiRequest) => {
        return {
            Data: {
                S3_BUCKET: req.body.bucket,
                S3_REGION: req.body.region,
            }
        };
    };

    mapDataToFields = (data: any) => {
        return {
            bucket: data.S3_BUCKET || "",
            region: data.S3_REGION || "",
        };
    };
}
