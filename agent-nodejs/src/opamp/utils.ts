import { AttributeValue, Attributes } from "@opentelemetry/api";
import { AnyValue, KeyValue } from "./generated/anyvalue_pb";

const attributeValueToAnyValue = (value: AttributeValue | undefined): AnyValue => {
    const anyValue = new AnyValue();
    if (typeof value === 'string') {
        anyValue.value = { value: value, case: "stringValue" };
    } else if (typeof value === 'number') {
        if (Number.isInteger(value)) {
            anyValue.value = { value: BigInt(value), case: "intValue" };
        } else {
            anyValue.value = { value, case: "doubleValue" };
        }
    } else if (typeof value === 'boolean') {
        anyValue.value = { value, case: "boolValue" };
    } else {
        // TODO: support this one day
        throw new Error(`Unsupported attribute value type: ${typeof value}`);
    }
    return anyValue;
}

export const otelAttributesToKeyValuePairs = (attributes?: Attributes): KeyValue[] | undefined => {
    if (!attributes) {
        return undefined;
    }
    return Object.entries(attributes)
        .filter(([_, value]) => value !== undefined) // Filter out attributes with undefined values
        .map(([key, value]) => {
        return new KeyValue({
            key,
            value: attributeValueToAnyValue(value),
        });
    });
};