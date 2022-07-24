import { Observable } from 'rxjs';
import { DataSourceInstanceSettings, ScopedVars, DataQueryRequest, DataQueryResponse } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { MongoDBDataSourceOptions, MongoDBQuery } from './types';

export class DataSource extends DataSourceWithBackend<MongoDBQuery, MongoDBDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MongoDBDataSourceOptions>) {
    super(instanceSettings);
  }

 applyTemplateVariables(query: MongoDBQuery, scopedVars: ScopedVars): Record<string, any> {
    const templateSrv = getTemplateSrv();
    return {
      ...query,
      aggregation: query.aggregation ? templateSrv.replace(query.aggregation, scopedVars) : ''
    };
  }

  query(request: DataQueryRequest<MongoDBQuery>): Observable<DataQueryResponse> {
      const templateSrv = getTemplateSrv();
      templateSrv.updateTimeRange(request.range);
      return super.query(request);
  }
}
