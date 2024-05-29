'use client';
import React, { useEffect, useState } from 'react';
import theme from '@/styles/palette';
import { ManagedSource } from '@/types';
import { useMutation } from 'react-query';
import { Back } from '@/assets/icons/overview';
import { DeleteSource } from '@/components/overview';
import { useKeyDown, useNotification } from '@/hooks';
import { useRouter, useSearchParams } from 'next/navigation';
import { deleteSource, getSource, patchSources } from '@/services';
import { ManageSourceHeader } from '@/components/overview/sources/manage.source.header/manage.source.header';
import {
  SETUP,
  ACTION,
  ROUTES,
  OVERVIEW,
  NOTIFICATION,
  LANGUAGES_LOGOS,
  getMainContainerLanguage,
} from '@/utils';
import {
  FieldWrapper,
  BackButtonWrapper,
  ManageSourcePageContainer,
  SaveSourceButtonWrapper,
} from './styled';
import {
  KeyvalButton,
  KeyvalInput,
  KeyvalLoader,
  KeyvalText,
  Conditions,
} from '@/design.system';

const NAME = 'name';
const KIND = 'kind';
const NAMESPACE = 'namespace';

export function EditSourceForm() {
  const [inputValue, setInputValue] = useState('');
  const [currentSource, setCurrentSource] = useState<ManagedSource>();

  const searchParams = useSearchParams();
  const router = useRouter();
  const { show, Notification } = useNotification();

  const { mutate: handleDeleteSource } = useMutation(() =>
    deleteSource(
      currentSource?.namespace || '',
      currentSource?.kind || '',
      currentSource?.name || ''
    )
  );

  const { mutate: editSource } = useMutation(() =>
    patchSources(
      currentSource?.namespace || '',
      currentSource?.kind || '',
      currentSource?.name || '',
      { reported_name: inputValue }
    )
  );
  useEffect(() => {
    onPageLoad();
  }, [searchParams]);

  useEffect(() => {
    setInputValue(currentSource?.reported_name || '');
  }, [currentSource]);

  useKeyDown('Enter', handleKeyPress);

  function handleKeyPress(e: any) {
    onSaveClick();
  }

  async function onPageLoad() {
    const name = searchParams.get(NAME) || '';
    const kind = searchParams.get(KIND) || '';
    const namespace = searchParams.get(NAMESPACE) || '';

    const currentSource = await getSource(namespace, kind, name);
    setCurrentSource(currentSource);
  }
  function onError({ response }) {
    const message = response?.data?.message;
    show({
      type: NOTIFICATION.ERROR,
      message,
    });
  }

  function onSaveClick() {
    editSource(undefined, {
      onError,
      onSuccess: () => router.push(`${ROUTES.SOURCES}?status=updated`),
    });
  }

  function onSourceDelete() {
    handleDeleteSource(undefined, {
      onSuccess: () => router.push(`${ROUTES.SOURCES}?status=deleted`),
      onError,
    });
  }

  if (!currentSource) {
    return <KeyvalLoader />;
  }

  return (
    <ManageSourcePageContainer>
      <BackButtonWrapper onClick={() => router.back()}>
        <Back width={14} />
        <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
      </BackButtonWrapper>
      {currentSource && <ManageSourceHeader source={currentSource} />}
      <div style={{ display: 'flex', gap: 60 }}>
        <div>
          <FieldWrapper>
            <KeyvalInput
              label={OVERVIEW.REPORTED_NAME}
              value={inputValue}
              onChange={(e) => setInputValue(e)}
            />
          </FieldWrapper>
          <SaveSourceButtonWrapper>
            <KeyvalButton disabled={!inputValue} onClick={onSaveClick}>
              <KeyvalText color={theme.colors.dark_blue} size={14} weight={600}>
                {ACTION.SAVE}
              </KeyvalText>
            </KeyvalButton>
          </SaveSourceButtonWrapper>
          <DeleteSource
            onDelete={onSourceDelete}
            name={currentSource?.name}
            image_url={
              LANGUAGES_LOGOS[
                getMainContainerLanguage(
                  currentSource?.instrumented_application_details?.languages ||
                    undefined
                )
              ]
            }
          />
        </div>
        <Conditions
          conditions={
            currentSource.instrumented_application_details?.conditions
          }
        />
      </div>

      <Notification />
    </ManageSourcePageContainer>
  );
}
