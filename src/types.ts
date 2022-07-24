import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface MongoDBQuery extends DataQuery {
  database: string;
  collection: string;
  timestampField: string;
  labelFields: string[];
  valueFields: string[];
  valueFieldTypes: string[];
  aggregation: string;
}

export const defaultQuery: Partial<MongoDBQuery> = {
    database: "my_db",
    collection: "my_collection",
    timestampField: "timestamp",
    labelFields: [ "sensorID" ],
    valueFields: [ "measurement" ],
    valueFieldTypes: [ "float64" ],
    aggregation: JSON.stringify([
        { 
            "$project": { 
                "timestamp": 1, 
                "sensorID": "$metadata.sensorID",
                "measurement": 1, 
                "_id": 0 
            }
        }
    ])
};

/**
 * These are options configured for each DataSource instance.
 */
export interface MongoDBDataSourceOptions extends DataSourceJsonData {
  url?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MongoDBSecureJsonData {
    username?: string;
    password?: string;
}
