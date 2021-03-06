import styled from "styled-components";

export const CenteredContainer = ({ className, children }) => {
  return <div className={className}>{children}</div>;
};

export const StyledCenteredContainer = styled(CenteredContainer)`
  display: flex;
  flex-direction: column;
  align-items: center;
`;

export const InputsWrapper = styled.div`
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: start;
  margin-bottom: 15px;
`;

export const WidthWrapper = styled.div`
  margin-top: 10px;
  max-width: 700px;
  width: 87%;
`;

export const DashBoardWrapper = styled.div`
  display: flex;
`;

export const ContentWrapper = styled.div`
  @media (max-width: 599px) {
    margin-top: 56px;
  }
  @media (min-width: 599px) {
    width: 64px;
  }
  margin-top: 64px;
  margin-bottom: 20px;
  flex-grow: 1;
`;

export const ContentSpacer = styled.div`
  width: 100%;
  height: 76px;
`;
