import React, { ChangeEvent, PureComponent } from 'react';
import {
  FieldSet,
  InlineField,
  InlineFieldRow,
  Input,
  SecretInput,
} from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MongoDBDataSourceOptions, MongoDBSecureJsonData } from './types';

interface Props extends DataSourcePluginOptionsEditorProps<MongoDBDataSourceOptions> {}

interface State {}

export class ConfigEditor extends PureComponent<Props, State> {
  onURLChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      url: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onUsernameChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const secureJsonData = {
      ...options.secureJsonData,
      username: event.target.value,
    };
    onOptionsChange({ ...options, secureJsonData });
  };
  onPasswordChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const secureJsonData = {
      ...options.secureJsonData,
      password: event.target.value,
    };
    onOptionsChange({ ...options, secureJsonData });
  };


  onResetCredential = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        username: false,
        password: false
      },
      secureJsonData: {
        ...options.secureJsonData,
        username: '',
        password: ''
      },
    });
  };

  readonly shortWidth = 15;
  readonly longWidth = 60;

  render() {
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as MongoDBSecureJsonData;

    return (
      <>
        <FieldSet label="MongoDB Connection" width={400}>
          <InlineField labelWidth={this.shortWidth} label="URL">
            <Input
              width={this.longWidth}
              name="url"
              type="text"
              onChange={this.onURLChange}
              value={jsonData.url || ''}
              placeholder="mongodb[+svc]://hostname:port[,hostname:port][/?key=value]"
            ></Input>
          </InlineField>
          <InlineFieldRow>
            <InlineField labelWidth={this.shortWidth} label="Username">
              <SecretInput
                width={this.shortWidth}
                isConfigured={(secureJsonFields && secureJsonFields.username) as boolean}
                value={secureJsonData.username || ''}
                placeholder="Username"
                onReset={this.onResetCredential}
                onChange={this.onUsernameChange}
              ></SecretInput>
            </InlineField>
            <InlineField label="Password" labelWidth={this.shortWidth}>
              <SecretInput
                width={this.shortWidth}
                isConfigured={(secureJsonFields && secureJsonFields.password) as boolean}
                value={secureJsonData.password || ''}
                label="Password"
                placeholder="Password"
                onReset={this.onResetCredential}
                onChange={this.onPasswordChange}
              ></SecretInput>
            </InlineField>
          </InlineFieldRow>
        </FieldSet>
      </>
    );
  }
}
