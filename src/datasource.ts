import { DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { MongoDBDataSourceOptions, MongoDBQuery } from './types';

export class DataSource extends DataSourceWithBackend<MongoDBQuery, MongoDBDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MongoDBDataSourceOptions>) {
    super(instanceSettings);
  }

 applyTemplateVariables(query: MongoDBQuery, scopedVars: ScopedVars): Record<string, any> {
    const templateSrv = getTemplateSrv();
    const newAggregation = query.aggregation ? templateSrv.replace(query.aggregation, scopedVars) : ''
    return {
      ...query,
      aggregation: newAggregation
    };
  }
}
