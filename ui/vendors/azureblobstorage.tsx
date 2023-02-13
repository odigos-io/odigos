import {
    ObservabilityVendor,
    ObservabilitySignals,
    VendorType,
} from "@/vendors/index";
import AzureBlobLogo from "@/img/vendor/blobstorage.svg";
import { NextApiRequest } from "next";

export class AzureBlobStorage implements ObservabilityVendor {
    name = "azureblob";
    displayName = "Azure Blob Storage";
    type = VendorType.MANAGED;
    supportedSignals = [ObservabilitySignals.Traces, ObservabilitySignals.Logs];
    getLogo = (props: any) => {
        return <AzureBlobLogo {...props} />;
    };

    getFields = (selectedSignals: any) => {
        return [
            {
                displayName: "Account Name",
                id: "account_name",
                name: "account_name",
                type: "text",
            },
            {
                displayName: "Container Name",
                id: "container",
                name: "container",
                type: "text",
            }
        ];
    };

    toObjects = (req: NextApiRequest) => {
        return {
            Data: {
                AZURE_BLOB_ACCOUNT_NAME: req.body.account_name,
                AZURE_BLOB_CONTAINER_NAME: req.body.container,
            }
        };
    };

    mapDataToFields = (data: any) => {
        return {
            account_name: data.AZURE_BLOB_ACCOUNT_NAME || "",
            container: data.AZURE_BLOB_CONTAINER_NAME || "",
        };
    };
}
