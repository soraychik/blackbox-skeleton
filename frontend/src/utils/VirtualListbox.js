import React from 'react';
import { FixedSizeList } from 'react-window';

const OuterElementContext = React.createContext({});

const OuterElementType = React.forwardRef((props, ref) => {
  const outerProps = React.useContext(OuterElementContext);
  return <div ref={ref} {...props} {...outerProps} />;
});

const renderRow = ({ data, index, style }) => {
  return React.cloneElement(data[index], { style });
};

const ListboxComponent = React.forwardRef(function ListboxComponent(props, ref) {
  const { children, ...other } = props;
  const itemData = React.Children.toArray(children);
  const itemCount = itemData.length;
  const itemSize = 46;
  const height = Math.min(8, itemCount) * itemSize;

  return (
    <div ref={ref}>
      <OuterElementContext.Provider value={other}>
        <FixedSizeList
          itemData={itemData}
          itemCount={itemCount}
          itemSize={itemSize}
          height={height}
          width="100%"
          outerElementType={OuterElementType}
          overscanCount={5}
        >
          {renderRow}
        </FixedSizeList>
      </OuterElementContext.Provider>
    </div>
  );
});

export default ListboxComponent;