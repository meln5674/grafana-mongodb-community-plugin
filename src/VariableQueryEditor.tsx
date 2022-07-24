import React, { useState } from 'react';
import { MongoDBVariableQuery } from './types';

interface VariableQueryProps {
  query: MongoDBVariableQuery;
  onChange: (query: MongoDBVariableQuery, definition: string) => void;
}

export const VariableQueryEditor: React.FC<VariableQueryProps> = ({ onChange, query }) => {
  const [state, setState] = useState(query);

  const saveQuery = () => {
    onChange(state, `${state.database} (${state.collection})`);
  };

  const handleChange = (event: React.FormEvent<HTMLInputElement>) =>
    setState({
      ...state,
      [event.currentTarget.name]: event.currentTarget.value,
    });

  return (
    <>
      <div className="gf-form">
        <span className="gf-form-label width-10">Database</span>
        <input
          name="database"
          className="gf-form-input"
          onBlur={saveQuery}
          onChange={handleChange}
          value={state.database}
        />
      </div>
      <div className="gf-form">
        <span className="gf-form-label width-10">Collection</span>
        <input
          name="collection"
          className="gf-form-input"
          onBlur={saveQuery}
          onChange={handleChange}
          value={state.collection}
        />
      </div>

      <div className="gf-form">
        <span className="gf-form-label width-10">Aggregation</span>
        <input
          name="aggregation"
          className="gf-form-input"
          onBlur={saveQuery}
          onChange={handleChange}
          value={state.aggregation}
        />
      </div>
      <div className="gf-form">
        <span className="gf-form-label width-10">Field Name</span>
        <input
          name="fieldName"
          className="gf-form-input"
          onBlur={saveQuery}
          onChange={handleChange}
          value={state.fieldName}
        />
      </div>
      <div className="gf-form">
        <span className="gf-form-label width-10">Field Type</span>
        <input
          name="fieldType"
          className="gf-form-input"
          onBlur={saveQuery}
          onChange={handleChange}
          value={state.fieldType}
        />
      </div>

    </>
  );
};
