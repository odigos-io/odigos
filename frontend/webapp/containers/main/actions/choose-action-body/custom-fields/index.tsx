import { ActionsType } from '@/types';
import AddClusterInfo from './add-cluster-info';
import DeleteAttributes from './delete-attributes';
import RenameAttributes from './rename-attributes';
import PiiMasking from './pii-masking';

interface ActionCustomFieldsProps {
  actionType?: ActionsType;
  value: string;
  setValue: (value: string) => void;
}

const ActionCustomFields: React.FC<ActionCustomFieldsProps> = ({ actionType, value, setValue }) => {
  switch (actionType) {
    case ActionsType.ADD_CLUSTER_INFO: {
      return <AddClusterInfo value={value} setValue={setValue} />;
    }

    case ActionsType.DELETE_ATTRIBUTES: {
      return <DeleteAttributes value={value} setValue={setValue} />;
    }

    case ActionsType.RENAME_ATTRIBUTES: {
      return <RenameAttributes value={value} setValue={setValue} />;
    }

    case ActionsType.PII_MASKING: {
      return <PiiMasking value={value} setValue={setValue} />;
    }

    case ActionsType.ERROR_SAMPLER:
      return null;

    case ActionsType.PROBABILISTIC_SAMPLER:
      return null;

    case ActionsType.LATENCY_SAMPLER:
      return null;

    default:
      return null;
  }
};

export default ActionCustomFields;
